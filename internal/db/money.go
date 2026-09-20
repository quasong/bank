package db

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"bank/internal/account"
	"bank/internal/currency"
	"bank/internal/ledger"
)

func (s *Store) OpenDeposit(ctx context.Context, acct account.Account) (account.Account, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return account.Account{}, err
	}
	defer tx.Rollback(ctx)

	if acct.Currency == "" {
		acct.Currency = currency.USD
	}
	const insertAcct = `
		INSERT INTO accounts (id, customer_id, currency, account_number, status, balance_cents, product, label)
		VALUES ($1, $2, $3, $4, $5, 0, $6, $7)
		RETURNING opened_at`
	if acct.Product == "" {
		acct.Product = account.ProductSpend
	}
	if err := tx.QueryRow(ctx, insertAcct, acct.ID, acct.CustomerID, string(acct.Currency), acct.AccountNumber, string(acct.Status), string(acct.Product), acct.Label).Scan(&acct.OpenedAt); err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			if strings.Contains(pe.ConstraintName, "spend") || strings.Contains(pe.ConstraintName, "customer_id") {
				return account.Account{}, account.ErrExists
			}
			if strings.Contains(pe.ConstraintName, "account_number") {
				return account.Account{}, account.ErrNumberTaken
			}
			return account.Account{}, err
		}
		return account.Account{}, err
	}
	const insertLedger = `
		INSERT INTO ledger_accounts (id, name, kind, currency, account_id)
		VALUES ($1, $2, $3, $4, $5)`
	if _, err := tx.Exec(ctx, insertLedger, acct.LedgerID, "Demand deposit "+acct.AccountNumber, string(ledger.KindLiability), string(acct.Currency), acct.ID); err != nil {
		return account.Account{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return account.Account{}, err
	}
	acct.BalanceCents = 0
	return acct, nil
}

const accountSelect = `SELECT a.id, a.customer_id, a.currency, a.account_number, a.status, a.balance_cents, a.opened_at, la.id, a.product, a.label
		FROM accounts a
		JOIN ledger_accounts la ON la.account_id = a.id`

func (s *Store) GetByID(ctx context.Context, id uuid.UUID) (account.Account, error) {
	return s.scanAccount(ctx, s.pool, accountSelect+` WHERE a.id = $1`, id)
}

func (s *Store) GetByCustomerCurrency(ctx context.Context, customerID uuid.UUID, ccy currency.Code) (account.Account, error) {
	if ccy == "" {
		ccy = currency.USD
	}
	return s.scanAccount(ctx, s.pool, accountSelect+` WHERE a.customer_id = $1 AND a.currency = $2 AND a.product = 'spend'`, customerID, string(ccy))
}

func (s *Store) GetByNumber(ctx context.Context, number string) (account.Account, error) {
	n, _, ok := currency.Normalize(number)
	if !ok {
		return account.Account{}, account.ErrNotFound
	}
	return s.scanAccount(ctx, s.pool, accountSelect+` WHERE a.account_number = $1`, n)
}

func (s *Store) ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]account.Account, error) {
	rows, err := s.pool.Query(ctx, accountSelect+` WHERE a.customer_id = $1`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []account.Account
	for rows.Next() {
		a, err := scanAccountRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if out == nil {
		out = []account.Account{}
	}
	sortAccounts(out)
	return out, rows.Err()
}

func (s *Store) ListAll(ctx context.Context) ([]account.Account, error) {
	rows, err := s.pool.Query(ctx, accountSelect+` ORDER BY a.opened_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []account.Account
	for rows.Next() {
		a, err := scanAccountRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if out == nil {
		out = []account.Account{}
	}
	return out, rows.Err()
}

func (s *Store) SetNumber(ctx context.Context, id uuid.UUID, number string) error {
	n, _, ok := currency.Normalize(number)
	if !ok {
		return account.ErrInvalidRequest
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE accounts SET account_number = $2 WHERE id = $1`, id, n)
	if err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			return account.ErrNumberTaken
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return account.ErrNotFound
	}
	if _, err := tx.Exec(ctx, `UPDATE ledger_accounts SET name = $2 WHERE account_id = $1`, id, "Demand deposit "+n); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) SetLabel(ctx context.Context, id uuid.UUID, label string) (account.Account, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE accounts SET label = $2 WHERE id = $1 AND product = 'jar' AND status <> 'closed'`, id, label)
	if err != nil {
		return account.Account{}, err
	}
	if tag.RowsAffected() == 0 {
		acct, getErr := s.GetByID(ctx, id)
		if getErr != nil {
			return account.Account{}, getErr
		}
		if !acct.IsJar() {
			return account.Account{}, account.ErrJar
		}
		if acct.Status == account.StatusClosed {
			return account.Account{}, account.ErrClosed
		}
		return account.Account{}, account.ErrNotFound
	}
	return s.GetByID(ctx, id)
}

func (s *Store) SetStatus(ctx context.Context, id uuid.UUID, next account.Status) (account.Account, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return account.Account{}, err
	}
	defer tx.Rollback(ctx)

	acct, err := s.scanAccount(ctx, tx, accountSelect+`
		WHERE a.id = $1
		FOR UPDATE OF a`, id)
	if err != nil {
		return account.Account{}, err
	}
	want, err := account.Transition(acct, next)
	if err != nil {
		return account.Account{}, err
	}
	if want != acct.Status {
		if _, err := tx.Exec(ctx, `UPDATE accounts SET status = $2 WHERE id = $1`, id, string(want)); err != nil {
			return account.Account{}, err
		}
		acct.Status = want
	}
	if err := tx.Commit(ctx); err != nil {
		return account.Account{}, err
	}
	return acct, nil
}

func (s *Store) Post(ctx context.Context, journal ledger.Journal, deltas map[uuid.UUID]int64) (ledger.Journal, bool, error) {
	if err := ledger.Validate(journal.Lines); err != nil {
		return ledger.Journal{}, false, err
	}
	if strings.TrimSpace(journal.IdempotencyKey) == "" {
		return ledger.Journal{}, false, ledger.ErrIdempotency
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ledger.Journal{}, false, err
	}
	defer tx.Rollback(ctx)

	ids := make([]uuid.UUID, 0, len(deltas))
	for id := range deltas {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	for _, id := range ids {
		var status string
		err := tx.QueryRow(ctx, `SELECT status FROM accounts WHERE id = $1 FOR UPDATE`, id).Scan(&status)
		if errors.Is(err, pgx.ErrNoRows) {
			return ledger.Journal{}, false, account.ErrNotFound
		}
		if err != nil {
			return ledger.Journal{}, false, err
		}
		if err := account.Status(status).MoneyError(); err != nil {
			return ledger.Journal{}, false, err
		}
	}

	const insertJ = `
		INSERT INTO journals (id, description, note, kind, idempotency_key)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at`
	err = tx.QueryRow(ctx, insertJ, journal.ID, journal.Description, journal.Note, string(journal.Kind), journal.IdempotencyKey).Scan(&journal.CreatedAt)
	if err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				return ledger.Journal{}, false, rollbackErr
			}
			existing, loadErr := s.journalByKey(ctx, journal.IdempotencyKey)
			return existing, true, loadErr
		}
		return ledger.Journal{}, false, err
	}

	const insertLine = `
		INSERT INTO journal_lines (id, journal_id, ledger_account_id, side, amount_cents)
		VALUES ($1, $2, $3, $4, $5)`
	for _, ln := range journal.Lines {
		if _, err := tx.Exec(ctx, insertLine, uuid.New(), journal.ID, ln.LedgerAccountID, string(ln.Side), ln.AmountCents); err != nil {
			return ledger.Journal{}, false, err
		}
	}
	for _, id := range ids {
		delta := deltas[id]
		tag, err := tx.Exec(ctx, `
			UPDATE accounts
			SET balance_cents = balance_cents + $2
			WHERE id = $1 AND status = 'active' AND balance_cents + $2 >= 0`, id, delta)
		if err != nil {
			return ledger.Journal{}, false, err
		}
		if tag.RowsAffected() == 0 {
			return ledger.Journal{}, false, account.ErrInsufficient
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return ledger.Journal{}, false, err
	}
	return journal, false, nil
}

func (s *Store) Activity(ctx context.Context, accountID uuid.UUID, limit, offset int32) ([]ledger.Entry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT j.id, j.created_at, j.kind, j.description, jl.side, jl.amount_cents, j.note,
			(
				SELECT a.account_number
				FROM journal_lines ojl
				JOIN ledger_accounts ola ON ola.id = ojl.ledger_account_id
				JOIN accounts a ON a.id = ola.account_id
				WHERE ojl.journal_id = jl.journal_id
				  AND ojl.id <> jl.id
				LIMIT 1
			)
		FROM journal_lines jl
		JOIN journals j ON j.id = jl.journal_id
		JOIN ledger_accounts la ON la.id = jl.ledger_account_id
		WHERE la.account_id = $1
		ORDER BY j.created_at DESC, jl.id DESC
		LIMIT $2 OFFSET $3`, accountID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ledger.Entry
	for rows.Next() {
		var e ledger.Entry
		var counterparty *string
		if err := rows.Scan(&e.JournalID, &e.CreatedAt, &e.Kind, &e.Description, &e.Side, &e.AmountCents, &e.Note, &counterparty); err != nil {
			return nil, err
		}
		if counterparty != nil {
			e.CounterpartyNumber = *counterparty
		}
		if e.Side == ledger.Credit {
			e.SignedCents = e.AmountCents
		} else {
			e.SignedCents = -e.AmountCents
		}
		out = append(out, e)
	}
	if out == nil {
		out = []ledger.Entry{}
	}
	return out, rows.Err()
}

func (s *Store) journalByKey(ctx context.Context, key string) (ledger.Journal, error) {
	var j ledger.Journal
	err := s.pool.QueryRow(ctx, `
		SELECT id, created_at, description, note, kind, idempotency_key
		FROM journals WHERE idempotency_key = $1`, key).Scan(
		&j.ID, &j.CreatedAt, &j.Description, &j.Note, &j.Kind, &j.IdempotencyKey,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ledger.Journal{}, ledger.ErrNotFound
	}
	if err != nil {
		return ledger.Journal{}, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT ledger_account_id, side, amount_cents
		FROM journal_lines WHERE journal_id = $1 ORDER BY id`, j.ID)
	if err != nil {
		return ledger.Journal{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var ln ledger.Line
		if err := rows.Scan(&ln.LedgerAccountID, &ln.Side, &ln.AmountCents); err != nil {
			return ledger.Journal{}, err
		}
		j.Lines = append(j.Lines, ln)
	}
	return j, rows.Err()
}

type rowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type accountRow interface {
	Scan(dest ...any) error
}

func (s *Store) scanAccount(ctx context.Context, q rowQuerier, sql string, args ...any) (account.Account, error) {
	a, err := scanAccountRow(q.QueryRow(ctx, sql, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return account.Account{}, account.ErrNotFound
	}
	if err != nil {
		return account.Account{}, err
	}
	return a, nil
}

func scanAccountRow(row accountRow) (account.Account, error) {
	var a account.Account
	var ccy, product string
	err := row.Scan(&a.ID, &a.CustomerID, &ccy, &a.AccountNumber, &a.Status, &a.BalanceCents, &a.OpenedAt, &a.LedgerID, &product, &a.Label)
	if err != nil {
		return account.Account{}, err
	}
	a.Currency = currency.Code(ccy)
	a.Product = account.Product(product)
	if a.Product == "" {
		a.Product = account.ProductSpend
	}
	return a, nil
}

func sortAccounts(out []account.Account) {
	sort.Slice(out, func(i, j int) bool {
		ri, rj := out[i].Currency.Rank(), out[j].Currency.Rank()
		if ri != rj {
			return ri < rj
		}
		if out[i].IsSpend() != out[j].IsSpend() {
			return out[i].IsSpend()
		}
		return out[i].OpenedAt.Before(out[j].OpenedAt)
	})
}

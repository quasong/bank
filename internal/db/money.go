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
	"bank/internal/ledger"
)

func (s *Store) OpenDeposit(ctx context.Context, acct account.Account) (account.Account, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return account.Account{}, err
	}
	defer tx.Rollback(ctx)

	const insertAcct = `
		INSERT INTO accounts (id, customer_id, account_number, status, balance_cents)
		VALUES ($1, $2, $3, $4, 0)
		RETURNING opened_at`
	if err := tx.QueryRow(ctx, insertAcct, acct.ID, acct.CustomerID, acct.AccountNumber, string(acct.Status)).Scan(&acct.OpenedAt); err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			if strings.Contains(pe.ConstraintName, "customer_id") {
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
		INSERT INTO ledger_accounts (id, name, kind, account_id)
		VALUES ($1, $2, $3, $4)`
	if _, err := tx.Exec(ctx, insertLedger, acct.LedgerID, "Demand deposit "+acct.AccountNumber, string(ledger.KindLiability), acct.ID); err != nil {
		return account.Account{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return account.Account{}, err
	}
	acct.BalanceCents = 0
	return acct, nil
}

func (s *Store) GetByID(ctx context.Context, id uuid.UUID) (account.Account, error) {
	return s.scanAccount(ctx, s.pool, `
		SELECT a.id, a.customer_id, a.account_number, a.status, a.balance_cents, a.opened_at, la.id
		FROM accounts a
		JOIN ledger_accounts la ON la.account_id = a.id
		WHERE a.id = $1`, id)
}

func (s *Store) GetByCustomer(ctx context.Context, customerID uuid.UUID) (account.Account, error) {
	return s.scanAccount(ctx, s.pool, `
		SELECT a.id, a.customer_id, a.account_number, a.status, a.balance_cents, a.opened_at, la.id
		FROM accounts a
		JOIN ledger_accounts la ON la.account_id = a.id
		WHERE a.customer_id = $1`, customerID)
}

func (s *Store) GetByNumber(ctx context.Context, number string) (account.Account, error) {
	return s.scanAccount(ctx, s.pool, `
		SELECT a.id, a.customer_id, a.account_number, a.status, a.balance_cents, a.opened_at, la.id
		FROM accounts a
		JOIN ledger_accounts la ON la.account_id = a.id
		WHERE a.account_number = $1`, number)
}

func (s *Store) ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]account.Account, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.id, a.customer_id, a.account_number, a.status, a.balance_cents, a.opened_at, la.id
		FROM accounts a
		JOIN ledger_accounts la ON la.account_id = a.id
		WHERE a.customer_id = $1
		ORDER BY a.opened_at`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []account.Account
	for rows.Next() {
		var a account.Account
		if err := rows.Scan(&a.ID, &a.CustomerID, &a.AccountNumber, &a.Status, &a.BalanceCents, &a.OpenedAt, &a.LedgerID); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if out == nil {
		out = []account.Account{}
	}
	return out, rows.Err()
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
		if status != string(account.StatusActive) {
			if status == string(account.StatusClosed) {
				return ledger.Journal{}, false, account.ErrClosed
			}
			return ledger.Journal{}, false, account.ErrFrozen
		}
	}

	const insertJ = `
		INSERT INTO journals (id, description, kind, idempotency_key)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at`
	err = tx.QueryRow(ctx, insertJ, journal.ID, journal.Description, string(journal.Kind), journal.IdempotencyKey).Scan(&journal.CreatedAt)
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
		SELECT j.id, j.created_at, j.kind, j.description, jl.side, jl.amount_cents
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
		if err := rows.Scan(&e.JournalID, &e.CreatedAt, &e.Kind, &e.Description, &e.Side, &e.AmountCents); err != nil {
			return nil, err
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
		SELECT id, created_at, description, kind, idempotency_key
		FROM journals WHERE idempotency_key = $1`, key).Scan(
		&j.ID, &j.CreatedAt, &j.Description, &j.Kind, &j.IdempotencyKey,
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

func (s *Store) scanAccount(ctx context.Context, q rowQuerier, sql string, arg any) (account.Account, error) {
	var a account.Account
	err := q.QueryRow(ctx, sql, arg).Scan(
		&a.ID, &a.CustomerID, &a.AccountNumber, &a.Status, &a.BalanceCents, &a.OpenedAt, &a.LedgerID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return account.Account{}, account.ErrNotFound
	}
	if err != nil {
		return account.Account{}, err
	}
	return a, nil
}

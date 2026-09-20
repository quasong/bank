package account

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

	"bank/internal/currency"
	"bank/internal/ledger"
)

type Store interface {
	OpenDeposit(ctx context.Context, acct Account) (Account, error)
	GetByID(ctx context.Context, id uuid.UUID) (Account, error)
	GetByCustomerCurrency(ctx context.Context, customerID uuid.UUID, ccy currency.Code) (Account, error)
	GetByNumber(ctx context.Context, number string) (Account, error)
	ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]Account, error)
	SetStatus(ctx context.Context, id uuid.UUID, next Status) (Account, error)
	SetLabel(ctx context.Context, id uuid.UUID, label string) (Account, error)
	Post(ctx context.Context, journal ledger.Journal, deltas map[uuid.UUID]int64) (ledger.Journal, bool, error)
	Activity(ctx context.Context, accountID uuid.UUID, limit, offset int32) ([]ledger.Entry, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Open(ctx context.Context, customerID uuid.UUID, codes ...currency.Code) (Account, error) {
	if customerID == uuid.Nil {
		return Account{}, ErrInvalidRequest
	}
	ccy := currency.USD
	if len(codes) > 0 && codes[0] != "" {
		ccy = codes[0]
	}
	if !ccy.Valid() {
		return Account{}, ErrInvalidRequest
	}
	if _, err := s.store.GetByCustomerCurrency(ctx, customerID, ccy); err == nil {
		return Account{}, ErrExists
	} else if err != ErrNotFound {
		return Account{}, err
	}
	var last error
	for i := 0; i < 5; i++ {
		core, err := s.uniqueCore(ctx, customerID)
		if err != nil {
			return Account{}, err
		}
		num, err := currency.Issue(ccy, core)
		if err != nil {
			return Account{}, err
		}
		acct := Account{
			ID:            uuid.New(),
			CustomerID:    customerID,
			Currency:      ccy,
			AccountNumber: num,
			Product:       ProductSpend,
			Status:        StatusActive,
			LedgerID:      uuid.New(),
		}
		opened, err := s.store.OpenDeposit(ctx, acct)
		if err == nil {
			return opened, nil
		}
		if err == ErrExists {
			return Account{}, ErrExists
		}
		if err != ErrNumberTaken {
			return Account{}, err
		}
		last = err
	}
	return Account{}, last
}

type numberRewriter interface {
	ListAll(ctx context.Context) ([]Account, error)
	SetNumber(ctx context.Context, id uuid.UUID, number string) error
}

// SplitSharedCores reissues wallets that reused another of the same customer's
// 8-digit local account, so USD / EUR / GBP suffixes are not identical.
func (s *Service) SplitSharedCores(ctx context.Context) (int, error) {
	ns, ok := s.store.(numberRewriter)
	if !ok {
		return 0, nil
	}
	all, err := ns.ListAll(ctx)
	if err != nil {
		return 0, err
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].CustomerID != all[j].CustomerID {
			return all[i].CustomerID.String() < all[j].CustomerID.String()
		}
		if !all[i].OpenedAt.Equal(all[j].OpenedAt) {
			return all[i].OpenedAt.Before(all[j].OpenedAt)
		}
		return all[i].Currency.Rank() < all[j].Currency.Rank()
	})
	used := map[uuid.UUID]map[string]struct{}{}
	n := 0
	for _, a := range all {
		core := currency.Core(a.AccountNumber)
		if core == "" {
			continue
		}
		if used[a.CustomerID] == nil {
			used[a.CustomerID] = map[string]struct{}{}
		}
		if _, taken := used[a.CustomerID][core]; !taken {
			used[a.CustomerID][core] = struct{}{}
			continue
		}
		var last error
		rewritten := false
		for i := 0; i < 8; i++ {
			next, err := s.uniqueCore(ctx, a.CustomerID)
			if err != nil {
				return n, err
			}
			if _, taken := used[a.CustomerID][next]; taken {
				continue
			}
			num, err := currency.Issue(a.Currency, next)
			if err != nil {
				return n, err
			}
			if _, err := s.store.GetByNumber(ctx, num); err == nil {
				continue
			} else if err != ErrNotFound {
				return n, err
			}
			if err := ns.SetNumber(ctx, a.ID, num); err != nil {
				last = err
				continue
			}
			used[a.CustomerID][next] = struct{}{}
			n++
			rewritten = true
			break
		}
		if !rewritten {
			if last == nil {
				last = ErrNumberTaken
			}
			return n, last
		}
	}
	return n, nil
}

func (s *Service) uniqueCore(ctx context.Context, customerID uuid.UUID) (string, error) {
	list, err := s.store.ListByCustomer(ctx, customerID)
	if err != nil {
		return "", err
	}
	taken := map[string]struct{}{}
	for _, a := range list {
		if c := currency.Core(a.AccountNumber); c != "" {
			taken[c] = struct{}{}
		}
	}
	for i := 0; i < 16; i++ {
		core, err := newAccountCore()
		if err != nil {
			return "", err
		}
		if _, ok := taken[core]; !ok {
			return core, nil
		}
	}
	return "", ErrNumberTaken
}

func (s *Service) OpenJar(ctx context.Context, customerID uuid.UUID, ccy currency.Code, label string) (Account, error) {
	if customerID == uuid.Nil {
		return Account{}, ErrInvalidRequest
	}
	if !ccy.Valid() {
		return Account{}, ErrInvalidRequest
	}
	label, err := NormalizeLabel(label)
	if err != nil {
		return Account{}, err
	}
	spend, err := s.store.GetByCustomerCurrency(ctx, customerID, ccy)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Account{}, ErrNeedSpend
		}
		return Account{}, err
	}
	if err := spend.Status.MoneyError(); err != nil {
		return Account{}, err
	}
	list, err := s.store.ListByCustomer(ctx, customerID)
	if err != nil {
		return Account{}, err
	}
	open := 0
	for _, a := range list {
		if a.IsJar() && a.Currency == ccy && a.Status != StatusClosed {
			open++
		}
	}
	if open >= MaxJarsPerCurrency {
		return Account{}, ErrJarLimit
	}
	var last error
	for i := 0; i < 5; i++ {
		core, err := s.uniqueCore(ctx, customerID)
		if err != nil {
			return Account{}, err
		}
		num, err := currency.Issue(ccy, core)
		if err != nil {
			return Account{}, err
		}
		acct := Account{
			ID:            uuid.New(),
			CustomerID:    customerID,
			Currency:      ccy,
			AccountNumber: num,
			Product:       ProductJar,
			Label:         label,
			Status:        StatusActive,
			LedgerID:      uuid.New(),
		}
		opened, err := s.store.OpenDeposit(ctx, acct)
		if err == nil {
			return opened, nil
		}
		if err == ErrExists {
			return Account{}, ErrExists
		}
		if err != ErrNumberTaken {
			return Account{}, err
		}
		last = err
	}
	return Account{}, last
}

type MoveResult struct {
	Journal     ledger.Journal
	From        Account
	To          Account
	AmountCents int64
	Idempotent  bool
}

func (s *Service) Move(ctx context.Context, customerID, fromID, toID uuid.UUID, amountCents int64, idempotencyKey, note string) (MoveResult, error) {
	if amountCents <= 0 {
		return MoveResult{}, ErrInvalidAmount
	}
	key := strings.TrimSpace(idempotencyKey)
	if key == "" {
		return MoveResult{}, ErrIdempotency
	}
	from, err := s.GetOwned(ctx, customerID, fromID)
	if err != nil {
		return MoveResult{}, err
	}
	to, err := s.GetOwned(ctx, customerID, toID)
	if err != nil {
		return MoveResult{}, err
	}
	if from.ID == to.ID {
		return MoveResult{}, ErrSameAccount
	}
	if from.Currency != to.Currency {
		return MoveResult{}, ErrCurrency
	}
	if from.IsSpend() && to.IsSpend() {
		return MoveResult{}, ErrInvalidRequest
	}
	note, err = ledger.NormalizeNote(note)
	if err != nil {
		return MoveResult{}, err
	}
	if err := from.Status.MoneyError(); err != nil {
		return MoveResult{}, err
	}
	if err := to.Status.MoneyError(); err != nil {
		return MoveResult{}, err
	}
	if from.BalanceCents < amountCents {
		return MoveResult{}, ErrInsufficient
	}
	j := ledger.Journal{
		ID:             uuid.New(),
		Description:    fmt.Sprintf("Move %d %s from %s to %s", amountCents, from.Currency, from.AccountNumber, to.AccountNumber),
		Note:           note,
		Kind:           ledger.KindMove,
		IdempotencyKey: key,
		Lines: []ledger.Line{
			{LedgerAccountID: from.LedgerID, Side: ledger.Debit, AmountCents: amountCents, Currency: from.Currency},
			{LedgerAccountID: to.LedgerID, Side: ledger.Credit, AmountCents: amountCents, Currency: to.Currency},
		},
	}
	posted, replay, err := s.store.Post(ctx, j, map[uuid.UUID]int64{
		from.ID: -amountCents,
		to.ID:   amountCents,
	})
	if err != nil {
		return MoveResult{}, err
	}
	fromFresh, err := s.store.GetByID(ctx, from.ID)
	if err != nil {
		return MoveResult{}, err
	}
	toFresh, err := s.store.GetByID(ctx, to.ID)
	if err != nil {
		return MoveResult{}, err
	}
	return MoveResult{Journal: posted, From: fromFresh, To: toFresh, AmountCents: amountCents, Idempotent: replay}, nil
}

func (s *Service) RenameJar(ctx context.Context, customerID, accountID uuid.UUID, label string) (Account, error) {
	acct, err := s.GetOwned(ctx, customerID, accountID)
	if err != nil {
		return Account{}, err
	}
	if !acct.IsJar() {
		return Account{}, ErrJar
	}
	if acct.Status == StatusClosed {
		return Account{}, ErrClosed
	}
	label, err = NormalizeLabel(label)
	if err != nil {
		return Account{}, err
	}
	return s.store.SetLabel(ctx, acct.ID, label)
}

func (s *Service) List(ctx context.Context, customerID uuid.UUID) ([]Account, error) {
	return s.store.ListByCustomer(ctx, customerID)
}

func (s *Service) GetOwned(ctx context.Context, customerID, accountID uuid.UUID) (Account, error) {
	acct, err := s.store.GetByID(ctx, accountID)
	if err != nil {
		return Account{}, err
	}
	if acct.CustomerID != customerID {
		return Account{}, ErrNotFound
	}
	return acct, nil
}

func (s *Service) Fund(ctx context.Context, customerID, accountID uuid.UUID, amountCents int64, idempotencyKey string) (Account, ledger.Journal, bool, error) {
	if amountCents <= 0 {
		return Account{}, ledger.Journal{}, false, ErrInvalidAmount
	}
	key := strings.TrimSpace(idempotencyKey)
	if key == "" {
		return Account{}, ledger.Journal{}, false, ErrIdempotency
	}
	acct, err := s.GetOwned(ctx, customerID, accountID)
	if err != nil {
		return Account{}, ledger.Journal{}, false, err
	}
	if acct.IsJar() {
		return Account{}, ledger.Journal{}, false, ErrJar
	}
	if err := acct.Status.MoneyError(); err != nil {
		return Account{}, ledger.Journal{}, false, err
	}
	j := ledger.Journal{
		ID:             uuid.New(),
		Description:    fmt.Sprintf("Demo funding %d %s to %s", amountCents, acct.Currency, acct.AccountNumber),
		Kind:           ledger.KindFunding,
		IdempotencyKey: key,
		Lines: []ledger.Line{
			{LedgerAccountID: currency.Vault(acct.Currency), Side: ledger.Debit, AmountCents: amountCents, Currency: acct.Currency},
			{LedgerAccountID: acct.LedgerID, Side: ledger.Credit, AmountCents: amountCents, Currency: acct.Currency},
		},
	}
	posted, replay, err := s.store.Post(ctx, j, map[uuid.UUID]int64{acct.ID: amountCents})
	if err != nil {
		return Account{}, ledger.Journal{}, false, err
	}
	fresh, err := s.store.GetByID(ctx, acct.ID)
	if err != nil {
		return Account{}, ledger.Journal{}, false, err
	}
	return fresh, posted, replay, nil
}

func (s *Service) Activity(ctx context.Context, customerID, accountID uuid.UUID, limit, offset int32) ([]ledger.Entry, error) {
	if _, err := s.GetOwned(ctx, customerID, accountID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.store.Activity(ctx, accountID, limit, offset)
}

func (s *Service) Freeze(ctx context.Context, customerID, accountID uuid.UUID) (Account, error) {
	return s.setOwnedStatus(ctx, customerID, accountID, StatusFrozen)
}

func (s *Service) Unfreeze(ctx context.Context, customerID, accountID uuid.UUID) (Account, error) {
	return s.setOwnedStatus(ctx, customerID, accountID, StatusActive)
}

func (s *Service) Close(ctx context.Context, customerID, accountID uuid.UUID) (Account, error) {
	return s.setOwnedStatus(ctx, customerID, accountID, StatusClosed)
}

func (s *Service) setOwnedStatus(ctx context.Context, customerID, accountID uuid.UUID, next Status) (Account, error) {
	acct, err := s.GetOwned(ctx, customerID, accountID)
	if err != nil {
		return Account{}, err
	}
	if next == StatusClosed && acct.IsSpend() {
		list, err := s.store.ListByCustomer(ctx, customerID)
		if err != nil {
			return Account{}, err
		}
		for _, a := range list {
			if a.IsJar() && a.Currency == acct.Currency && a.Status != StatusClosed {
				return Account{}, ErrHasJars
			}
		}
	}
	return s.store.SetStatus(ctx, accountID, next)
}

func (s *Service) Withdraw(ctx context.Context, customerID, accountID uuid.UUID, amountCents int64, idempotencyKey string) (Account, ledger.Journal, bool, error) {
	if amountCents <= 0 {
		return Account{}, ledger.Journal{}, false, ErrInvalidAmount
	}
	key := strings.TrimSpace(idempotencyKey)
	if key == "" {
		return Account{}, ledger.Journal{}, false, ErrIdempotency
	}
	acct, err := s.GetOwned(ctx, customerID, accountID)
	if err != nil {
		return Account{}, ledger.Journal{}, false, err
	}
	if acct.IsJar() {
		return Account{}, ledger.Journal{}, false, ErrJar
	}
	if err := acct.Status.MoneyError(); err != nil {
		return Account{}, ledger.Journal{}, false, err
	}
	j := ledger.Journal{
		ID:             uuid.New(),
		Description:    fmt.Sprintf("Withdrawal %d %s from %s", amountCents, acct.Currency, acct.AccountNumber),
		Kind:           ledger.KindWithdrawal,
		IdempotencyKey: key,
		Lines: []ledger.Line{
			{LedgerAccountID: acct.LedgerID, Side: ledger.Debit, AmountCents: amountCents, Currency: acct.Currency},
			{LedgerAccountID: currency.Vault(acct.Currency), Side: ledger.Credit, AmountCents: amountCents, Currency: acct.Currency},
		},
	}
	posted, replay, err := s.store.Post(ctx, j, map[uuid.UUID]int64{acct.ID: -amountCents})
	if err != nil {
		return Account{}, ledger.Journal{}, false, err
	}
	fresh, err := s.store.GetByID(ctx, acct.ID)
	if err != nil {
		return Account{}, ledger.Journal{}, false, err
	}
	return fresh, posted, replay, nil
}

func newAccountCore() (string, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	n := binary.BigEndian.Uint64(buf[:]) % 100_000_000
	return fmt.Sprintf("%08d", n), nil
}

func ValidAccountNumber(n string) bool {
	_, _, ok := currency.Normalize(n)
	return ok
}

package account

import (
	"context"
	"crypto/rand"
	"encoding/binary"
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
	if _, err := s.GetOwned(ctx, customerID, accountID); err != nil {
		return Account{}, err
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

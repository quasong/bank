package account

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"bank/internal/ledger"
)

type Store interface {
	OpenDeposit(ctx context.Context, acct Account) (Account, error)
	GetByID(ctx context.Context, id uuid.UUID) (Account, error)
	GetByCustomer(ctx context.Context, customerID uuid.UUID) (Account, error)
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

func (s *Service) Open(ctx context.Context, customerID uuid.UUID) (Account, error) {
	if customerID == uuid.Nil {
		return Account{}, ErrInvalidRequest
	}
	if _, err := s.store.GetByCustomer(ctx, customerID); err == nil {
		return Account{}, ErrExists
	} else if err != ErrNotFound {
		return Account{}, err
	}
	var last error
	for i := 0; i < 5; i++ {
		num, err := newAccountNumber()
		if err != nil {
			return Account{}, err
		}
		acct := Account{
			ID:            uuid.New(),
			CustomerID:    customerID,
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
		Description:    fmt.Sprintf("Demo funding %d cents to %s", amountCents, acct.AccountNumber),
		Kind:           ledger.KindFunding,
		IdempotencyKey: key,
		Lines: []ledger.Line{
			{LedgerAccountID: ledger.VaultID, Side: ledger.Debit, AmountCents: amountCents},
			{LedgerAccountID: acct.LedgerID, Side: ledger.Credit, AmountCents: amountCents},
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
		Description:    fmt.Sprintf("Withdrawal %d cents from %s", amountCents, acct.AccountNumber),
		Kind:           ledger.KindWithdrawal,
		IdempotencyKey: key,
		Lines: []ledger.Line{
			{LedgerAccountID: acct.LedgerID, Side: ledger.Debit, AmountCents: amountCents},
			{LedgerAccountID: ledger.VaultID, Side: ledger.Credit, AmountCents: amountCents},
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

func newAccountNumber() (string, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	n := binary.BigEndian.Uint64(buf[:]) % 100_000_000
	return fmt.Sprintf("%08d", n), nil
}

func ValidAccountNumber(n string) bool {
	if len(n) != 8 {
		return false
	}
	for i := 0; i < 8; i++ {
		if n[i] < '0' || n[i] > '9' {
			return false
		}
	}
	return true
}

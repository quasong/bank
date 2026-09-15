package payee

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"bank/internal/account"
	"bank/internal/currency"
)

type Store interface {
	ListPayees(ctx context.Context, customerID uuid.UUID) ([]Payee, error)
	UpsertPayee(ctx context.Context, customerID uuid.UUID, accountNumber, displayName string, overwriteName bool) (Payee, error)
	RenamePayee(ctx context.Context, customerID, id uuid.UUID, displayName string, provided bool) (Payee, error)
	DeletePayee(ctx context.Context, customerID, id uuid.UUID) error
}

type Directory interface {
	GetByNumber(ctx context.Context, number string) (account.Account, error)
}

type Service struct {
	store    Store
	accounts Directory
}

func NewService(store Store, accounts Directory) *Service {
	return &Service{store: store, accounts: accounts}
}

func (s *Service) List(ctx context.Context, customerID uuid.UUID) ([]Payee, error) {
	if customerID == uuid.Nil {
		return nil, ErrInvalidRequest
	}
	out, err := s.store.ListPayees(ctx, customerID)
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = []Payee{}
	}
	return out, nil
}

func (s *Service) Names(ctx context.Context, customerID uuid.UUID) (map[string]string, error) {
	list, err := s.List(ctx, customerID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(list))
	for _, p := range list {
		out[p.AccountNumber] = p.DisplayName
	}
	return out, nil
}

func (s *Service) Upsert(ctx context.Context, customerID uuid.UUID, accountNumber, displayName string) (Payee, error) {
	if customerID == uuid.Nil {
		return Payee{}, ErrInvalidRequest
	}
	accountNumber = strings.TrimSpace(accountNumber)
	canonical, _, ok := currency.Normalize(accountNumber)
	if !ok {
		return Payee{}, ErrInvalidRequest
	}
	dest, err := s.accounts.GetByNumber(ctx, canonical)
	if err != nil {
		return Payee{}, err
	}
	if dest.CustomerID == customerID {
		return Payee{}, ErrOwnAccount
	}
	name, provided, err := normalizeName(displayName)
	if err != nil {
		return Payee{}, err
	}
	if !provided {
		name = dest.AccountNumber
	}
	return s.store.UpsertPayee(ctx, customerID, dest.AccountNumber, name, provided)
}

func (s *Service) Rename(ctx context.Context, customerID, id uuid.UUID, displayName string) (Payee, error) {
	if customerID == uuid.Nil || id == uuid.Nil {
		return Payee{}, ErrInvalidRequest
	}
	name, provided, err := normalizeName(displayName)
	if err != nil {
		return Payee{}, err
	}
	return s.store.RenamePayee(ctx, customerID, id, name, provided)
}

func (s *Service) Delete(ctx context.Context, customerID, id uuid.UUID) error {
	if customerID == uuid.Nil || id == uuid.Nil {
		return ErrInvalidRequest
	}
	return s.store.DeletePayee(ctx, customerID, id)
}

func normalizeName(s string) (string, bool, error) {
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return "", false, nil
	}
	if utf8.RuneCountInString(s) > MaxNameLen {
		return "", false, ErrInvalidRequest
	}
	return s, true, nil
}

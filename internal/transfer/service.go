package transfer

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"bank/internal/account"
	"bank/internal/currency"
	"bank/internal/ledger"
)

type Result struct {
	Journal     ledger.Journal
	From        account.Account
	To          account.Account
	AmountCents int64
	Idempotent  bool
}

type Service struct {
	store account.Store
}

func NewService(store account.Store) *Service {
	return &Service{store: store}
}

func (s *Service) Execute(ctx context.Context, customerID, fromID uuid.UUID, toNumber string, amountCents int64, idempotencyKey, note string) (Result, error) {
	if amountCents <= 0 {
		return Result{}, account.ErrInvalidAmount
	}
	key := strings.TrimSpace(idempotencyKey)
	if key == "" {
		return Result{}, account.ErrIdempotency
	}
	toNumber = strings.TrimSpace(toNumber)
	canonical, destCCY, ok := currency.Normalize(toNumber)
	if !ok {
		return Result{}, account.ErrInvalidRequest
	}

	from, err := s.store.GetByID(ctx, fromID)
	if err != nil {
		return Result{}, err
	}
	if from.CustomerID != customerID {
		return Result{}, account.ErrNotFound
	}
	to, err := s.store.GetByNumber(ctx, canonical)
	if err != nil {
		return Result{}, err
	}
	if from.ID == to.ID {
		return Result{}, account.ErrSameAccount
	}
	if from.Currency != to.Currency || from.Currency != destCCY {
		return Result{}, account.ErrCurrency
	}
	note, err = ledger.NormalizeNote(note)
	if err != nil {
		return Result{}, err
	}
	if err := from.Status.MoneyError(); err != nil {
		return Result{}, err
	}
	if err := to.Status.MoneyError(); err != nil {
		return Result{}, err
	}
	if from.BalanceCents < amountCents {
		return Result{}, account.ErrInsufficient
	}

	j := ledger.Journal{
		ID:             uuid.New(),
		Description:    fmt.Sprintf("Transfer %d %s from %s to %s", amountCents, from.Currency, from.AccountNumber, to.AccountNumber),
		Note:           note,
		Kind:           ledger.KindTransfer,
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
		return Result{}, err
	}
	fromFresh, err := s.store.GetByID(ctx, from.ID)
	if err != nil {
		return Result{}, err
	}
	toFresh, err := s.store.GetByID(ctx, to.ID)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Journal:     posted,
		From:        fromFresh,
		To:          toFresh,
		AmountCents: amountCents,
		Idempotent:  replay,
	}, nil
}

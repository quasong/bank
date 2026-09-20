package fx

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"bank/internal/account"
	"bank/internal/currency"
	"bank/internal/ledger"
)

type Quote struct {
	From        currency.Code
	To          currency.Code
	AmountCents int64
	QuoteCents  int64
	RateE8      int64
	Rate        string
	AsOf        string
}

type Result struct {
	Journal    ledger.Journal
	From       account.Account
	To         account.Account
	FromCents  int64
	ToCents    int64
	RateE8     int64
	Rate       string
	AsOf       string
	Idempotent bool
}

type Rates interface {
	Snapshot(ctx context.Context) (*Table, error)
}

type Service struct {
	accounts *account.Service
	store    account.Store
	rates    Rates
}

func NewService(accounts *account.Service, store account.Store, rates Rates) *Service {
	return &Service{accounts: accounts, store: store, rates: rates}
}

func (s *Service) Quote(ctx context.Context, from, to currency.Code, amountCents int64) (Quote, error) {
	if !from.Valid() || !to.Valid() || from == to {
		return Quote{}, account.ErrInvalidRequest
	}
	if amountCents < 0 {
		return Quote{}, account.ErrInvalidAmount
	}
	table, err := s.rates.Snapshot(ctx)
	if err != nil {
		return Quote{}, account.ErrRates
	}
	fromPer, okFrom := table.PerUSD[from]
	toPer, okTo := table.PerUSD[to]
	if !okFrom || !okTo {
		return Quote{}, account.ErrRates
	}
	rate, err := currency.CrossRateE8(fromPer, toPer)
	if err != nil {
		return Quote{}, account.ErrRates
	}
	q := Quote{
		From:        from,
		To:          to,
		AmountCents: amountCents,
		RateE8:      rate,
		Rate:        currency.FormatRate(rate),
		AsOf:        table.Date,
	}
	if amountCents > 0 {
		out, err := currency.ConvertCents(amountCents, rate)
		if err != nil {
			return Quote{}, account.ErrInvalidAmount
		}
		q.QuoteCents = out
	}
	return q, nil
}

func (s *Service) Convert(ctx context.Context, customerID, fromID uuid.UUID, toCCY currency.Code, amountCents int64, idempotencyKey string) (Result, error) {
	if amountCents <= 0 {
		return Result{}, account.ErrInvalidAmount
	}
	key := strings.TrimSpace(idempotencyKey)
	if key == "" {
		return Result{}, account.ErrIdempotency
	}
	if customerID == uuid.Nil || fromID == uuid.Nil || !toCCY.Valid() {
		return Result{}, account.ErrInvalidRequest
	}
	from, err := s.accounts.GetOwned(ctx, customerID, fromID)
	if err != nil {
		return Result{}, err
	}
	if from.Currency == toCCY {
		return Result{}, account.ErrInvalidRequest
	}
	if from.IsJar() {
		return Result{}, account.ErrJar
	}
	if err := from.Status.MoneyError(); err != nil {
		return Result{}, err
	}
	q, err := s.Quote(ctx, from.Currency, toCCY, amountCents)
	if err != nil {
		return Result{}, err
	}
	if q.QuoteCents <= 0 {
		return Result{}, account.ErrInvalidAmount
	}
	dest, err := s.ensurePocket(ctx, customerID, toCCY)
	if err != nil {
		return Result{}, err
	}
	if err := dest.Status.MoneyError(); err != nil {
		return Result{}, err
	}

	j := ledger.Journal{
		ID:             uuid.New(),
		Description:    fmt.Sprintf("FX %d %s to %d %s at %s", amountCents, from.Currency, q.QuoteCents, dest.Currency, q.Rate),
		Kind:           ledger.KindFX,
		IdempotencyKey: key,
		Lines: []ledger.Line{
			{LedgerAccountID: from.LedgerID, Side: ledger.Debit, AmountCents: amountCents, Currency: from.Currency},
			{LedgerAccountID: currency.Vault(from.Currency), Side: ledger.Credit, AmountCents: amountCents, Currency: from.Currency},
			{LedgerAccountID: currency.Vault(dest.Currency), Side: ledger.Debit, AmountCents: q.QuoteCents, Currency: dest.Currency},
			{LedgerAccountID: dest.LedgerID, Side: ledger.Credit, AmountCents: q.QuoteCents, Currency: dest.Currency},
		},
	}
	posted, replay, err := s.store.Post(ctx, j, map[uuid.UUID]int64{
		from.ID: -amountCents,
		dest.ID: q.QuoteCents,
	})
	if err != nil {
		return Result{}, err
	}
	fromFresh, err := s.store.GetByID(ctx, from.ID)
	if err != nil {
		return Result{}, err
	}
	toFresh, err := s.store.GetByID(ctx, dest.ID)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Journal:    posted,
		From:       fromFresh,
		To:         toFresh,
		FromCents:  amountCents,
		ToCents:    q.QuoteCents,
		RateE8:     q.RateE8,
		Rate:       q.Rate,
		AsOf:       q.AsOf,
		Idempotent: replay,
	}, nil
}

func (s *Service) ensurePocket(ctx context.Context, customerID uuid.UUID, ccy currency.Code) (account.Account, error) {
	dest, err := s.store.GetByCustomerCurrency(ctx, customerID, ccy)
	if err == nil {
		return dest, nil
	}
	if !errors.Is(err, account.ErrNotFound) {
		return account.Account{}, err
	}
	opened, err := s.accounts.Open(ctx, customerID, ccy)
	if err == nil {
		return opened, nil
	}
	if errors.Is(err, account.ErrExists) {
		return s.store.GetByCustomerCurrency(ctx, customerID, ccy)
	}
	return account.Account{}, err
}

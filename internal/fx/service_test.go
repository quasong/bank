package fx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"bank/internal/account"
	"bank/internal/currency"
	"bank/internal/ledger"
)

func testRates(t *testing.T) (*httptest.Server, *Client) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"amount":1.0,"base":"USD","date":"2026-09-14","rates":{"EUR":0.85,"GBP":0.75,"AUD":1.5}}`))
	}))
	t.Cleanup(srv.Close)
	return srv, NewClient(srv.URL, srv.Client())
}

func TestQuoteUSDToEUR(t *testing.T) {
	_, client := testRates(t)
	svc := NewService(account.NewService(account.NewMemStore()), account.NewMemStore(), client)
	q, err := svc.Quote(context.Background(), currency.USD, currency.EUR, 10000)
	if err != nil {
		t.Fatal(err)
	}
	if q.QuoteCents != 8500 || q.Rate != "0.85" || q.AsOf != "2026-09-14" {
		t.Fatalf("%+v", q)
	}
}

func TestQuoteUSDToAUD(t *testing.T) {
	_, client := testRates(t)
	svc := NewService(account.NewService(account.NewMemStore()), account.NewMemStore(), client)
	q, err := svc.Quote(context.Background(), currency.USD, currency.AUD, 10000)
	if err != nil {
		t.Fatal(err)
	}
	if q.QuoteCents != 15000 || q.Rate != "1.5" {
		t.Fatalf("%+v", q)
	}
}

func TestConvertOpensDestinationAndBalances(t *testing.T) {
	ctx := context.Background()
	store := account.NewMemStore()
	accounts := account.NewService(store)
	_, client := testRates(t)
	svc := NewService(accounts, store, client)

	cid := uuid.New()
	usd, err := accounts.Open(ctx, cid)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := accounts.Fund(ctx, cid, usd.ID, 10000, "in"); err != nil {
		t.Fatal(err)
	}
	res, err := svc.Convert(ctx, cid, usd.ID, currency.EUR, 10000, "fx-1")
	if err != nil {
		t.Fatal(err)
	}
	if res.From.BalanceCents != 0 || res.To.BalanceCents != 8500 || res.To.Currency != currency.EUR {
		t.Fatalf("%+v %+v", res.From, res.To)
	}
	if res.Journal.Kind != ledger.KindFX {
		t.Fatalf("kind %s", res.Journal.Kind)
	}
	replay, err := svc.Convert(ctx, cid, usd.ID, currency.EUR, 10000, "fx-1")
	if err != nil || !replay.Idempotent || replay.From.BalanceCents != 0 {
		t.Fatalf("replay %+v %v", replay, err)
	}
	fromAct, err := accounts.Activity(ctx, cid, usd.ID, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range fromAct {
		if e.Kind == ledger.KindFX && e.SignedCents == -10000 && e.CounterpartyNumber == res.To.AccountNumber {
			found = true
		}
	}
	if !found {
		t.Fatalf("usd activity %+v", fromAct)
	}
}

func TestConvertRejectsSameCurrency(t *testing.T) {
	ctx := context.Background()
	store := account.NewMemStore()
	accounts := account.NewService(store)
	_, client := testRates(t)
	svc := NewService(accounts, store, client)
	cid := uuid.New()
	usd, err := accounts.Open(ctx, cid)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Convert(ctx, cid, usd.ID, currency.USD, 100, "fx")
	if !errors.Is(err, account.ErrInvalidRequest) {
		t.Fatalf("got %v", err)
	}
}

func TestQuoteSameCurrencyRejected(t *testing.T) {
	_, client := testRates(t)
	svc := NewService(account.NewService(account.NewMemStore()), account.NewMemStore(), client)
	_, err := svc.Quote(context.Background(), currency.USD, currency.USD, 100)
	if !errors.Is(err, account.ErrInvalidRequest) {
		t.Fatalf("got %v", err)
	}
}

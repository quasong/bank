package account

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"bank/internal/currency"
	"bank/internal/ledger"
)

func TestValidAccountNumber(t *testing.T) {
	if !ValidAccountNumber("01234567") || ValidAccountNumber("TB012345") || ValidAccountNumber("1234567") || ValidAccountNumber("123456789") {
		t.Fatal("usd format")
	}
	gbp, err := currency.Issue(currency.GBP, "01234567")
	if err != nil || !ValidAccountNumber(gbp) {
		t.Fatal("gbp")
	}
	eur, err := currency.Issue(currency.EUR, "01234567")
	if err != nil || !ValidAccountNumber(eur) || !ValidAccountNumber(currency.Format(eur)) {
		t.Fatal("eur")
	}
}

func TestOpenSecondCurrencySharesCore(t *testing.T) {
	ctx := context.Background()
	store := NewMemStore()
	svc := NewService(store)
	cid := uuid.New()
	usd, err := svc.Open(ctx, cid)
	if err != nil {
		t.Fatal(err)
	}
	eur, err := svc.Open(ctx, cid, currency.EUR)
	if err != nil {
		t.Fatal(err)
	}
	if usd.Currency != currency.USD || eur.Currency != currency.EUR {
		t.Fatalf("%s %s", usd.Currency, eur.Currency)
	}
	if currency.Core(usd.AccountNumber) != currency.Core(eur.AccountNumber) {
		t.Fatalf("core %s vs %s", usd.AccountNumber, eur.AccountNumber)
	}
	if _, err := svc.Open(ctx, cid, currency.EUR); !errors.Is(err, ErrExists) {
		t.Fatalf("dup eur: %v", err)
	}
}

func TestOpenAndFund(t *testing.T) {
	ctx := context.Background()
	store := NewMemStore()
	svc := NewService(store)
	cid := uuid.New()

	acct, err := svc.Open(ctx, cid)
	if err != nil {
		t.Fatal(err)
	}
	if acct.BalanceCents != 0 || !ValidAccountNumber(acct.AccountNumber) {
		t.Fatalf("%+v", acct)
	}
	if _, err := svc.Open(ctx, cid); !errors.Is(err, ErrExists) {
		t.Fatalf("dup: %v", err)
	}

	funded, journal, replay, err := svc.Fund(ctx, cid, acct.ID, 5000, "fund-1")
	if err != nil || replay || journal.Kind != ledger.KindFunding {
		t.Fatalf("fund %v replay=%v err=%v", funded, replay, err)
	}
	if funded.BalanceCents != 5000 {
		t.Fatalf("balance %d", funded.BalanceCents)
	}
	again, _, replay, err := svc.Fund(ctx, cid, acct.ID, 5000, "fund-1")
	if err != nil || !replay {
		t.Fatalf("replay err=%v replay=%v", err, replay)
	}
	if again.BalanceCents != 5000 {
		t.Fatalf("replay doubled balance: %d", again.BalanceCents)
	}
}

func TestFundFrozen(t *testing.T) {
	ctx := context.Background()
	store := NewMemStore()
	svc := NewService(store)
	cid := uuid.New()
	acct, err := svc.Open(ctx, cid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Freeze(ctx, cid, acct.ID); err != nil {
		t.Fatal(err)
	}
	_, _, _, err = svc.Fund(ctx, cid, acct.ID, 100, "k")
	if !errors.Is(err, ErrFrozen) {
		t.Fatalf("got %v", err)
	}
}

func TestActivityFromJournal(t *testing.T) {
	ctx := context.Background()
	store := NewMemStore()
	svc := NewService(store)
	cid := uuid.New()
	acct, err := svc.Open(ctx, cid)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := svc.Fund(ctx, cid, acct.ID, 100, "k"); err != nil {
		t.Fatal(err)
	}
	items, err := svc.Activity(ctx, cid, acct.ID, 50, 0)
	if err != nil || len(items) != 1 || items[0].SignedCents != 100 {
		t.Fatalf("%+v %v", items, err)
	}
	if items[0].CounterpartyNumber != "" {
		t.Fatalf("funding should have no counterparty: %+v", items[0])
	}
}

func TestWithdrawAndIdempotentReplay(t *testing.T) {
	ctx := context.Background()
	store := NewMemStore()
	svc := NewService(store)
	cid := uuid.New()
	acct, err := svc.Open(ctx, cid)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := svc.Fund(ctx, cid, acct.ID, 5000, "in"); err != nil {
		t.Fatal(err)
	}
	out, journal, replay, err := svc.Withdraw(ctx, cid, acct.ID, 2000, "out-1")
	if err != nil || replay || journal.Kind != ledger.KindWithdrawal {
		t.Fatalf("withdraw %+v replay=%v err=%v", journal, replay, err)
	}
	if out.BalanceCents != 3000 {
		t.Fatalf("balance %d", out.BalanceCents)
	}
	again, _, replay, err := svc.Withdraw(ctx, cid, acct.ID, 2000, "out-1")
	if err != nil || !replay {
		t.Fatalf("replay err=%v replay=%v", err, replay)
	}
	if again.BalanceCents != 3000 {
		t.Fatalf("replay doubled withdrawal: %d", again.BalanceCents)
	}
	items, err := svc.Activity(ctx, cid, acct.ID, 50, 0)
	if err != nil || len(items) != 2 {
		t.Fatalf("activity %+v %v", items, err)
	}
}

func TestFreezeUnfreezeClose(t *testing.T) {
	ctx := context.Background()
	store := NewMemStore()
	svc := NewService(store)
	cid := uuid.New()
	acct, err := svc.Open(ctx, cid)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := svc.Fund(ctx, cid, acct.ID, 100, "in"); err != nil {
		t.Fatal(err)
	}

	frozen, err := svc.Freeze(ctx, cid, acct.ID)
	if err != nil || frozen.Status != StatusFrozen {
		t.Fatalf("%+v %v", frozen, err)
	}
	again, err := svc.Freeze(ctx, cid, acct.ID)
	if err != nil || again.Status != StatusFrozen {
		t.Fatalf("idempotent freeze %v", err)
	}
	if _, _, _, err := svc.Withdraw(ctx, cid, acct.ID, 50, "out"); !errors.Is(err, ErrFrozen) {
		t.Fatalf("withdraw frozen: %v", err)
	}

	active, err := svc.Unfreeze(ctx, cid, acct.ID)
	if err != nil || active.Status != StatusActive {
		t.Fatalf("%+v %v", active, err)
	}
	if _, err := svc.Close(ctx, cid, acct.ID); !errors.Is(err, ErrHasBalance) {
		t.Fatalf("close with balance: %v", err)
	}

	if _, _, _, err := svc.Withdraw(ctx, cid, acct.ID, 100, "drain"); err != nil {
		t.Fatal(err)
	}
	closed, err := svc.Close(ctx, cid, acct.ID)
	if err != nil || closed.Status != StatusClosed {
		t.Fatalf("%+v %v", closed, err)
	}
	if _, err := svc.Unfreeze(ctx, cid, acct.ID); !errors.Is(err, ErrClosed) {
		t.Fatalf("unfreeze closed: %v", err)
	}
	if _, _, _, err := svc.Fund(ctx, cid, acct.ID, 1, "after-close"); !errors.Is(err, ErrClosed) {
		t.Fatalf("fund closed: %v", err)
	}
}

func TestTransition(t *testing.T) {
	zero := Account{Status: StatusActive, BalanceCents: 0}
	got, err := Transition(zero, StatusClosed)
	if err != nil || got != StatusClosed {
		t.Fatalf("close zero: %v %v", got, err)
	}
	funded := Account{Status: StatusActive, BalanceCents: 1}
	if _, err := Transition(funded, StatusClosed); !errors.Is(err, ErrHasBalance) {
		t.Fatalf("got %v", err)
	}
	closed := Account{Status: StatusClosed}
	if _, err := Transition(closed, StatusActive); !errors.Is(err, ErrClosed) {
		t.Fatalf("got %v", err)
	}
}

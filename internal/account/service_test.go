package account

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"bank/internal/ledger"
)

func TestValidAccountNumber(t *testing.T) {
	if !ValidAccountNumber("01234567") || ValidAccountNumber("TB012345") || ValidAccountNumber("1234567") || ValidAccountNumber("123456789") {
		t.Fatal("expected exactly eight digits")
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
	store.Freeze(acct.ID)
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
}

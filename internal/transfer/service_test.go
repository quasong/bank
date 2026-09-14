package transfer

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"bank/internal/account"
)

func TestTransferMovesBalancesAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	store := account.NewMemStore()
	accounts := account.NewService(store)
	svc := NewService(store)

	aCust := uuid.New()
	bCust := uuid.New()
	from, err := accounts.Open(ctx, aCust)
	if err != nil {
		t.Fatal(err)
	}
	to, err := accounts.Open(ctx, bCust)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := accounts.Fund(ctx, aCust, from.ID, 1000, "fund-a"); err != nil {
		t.Fatal(err)
	}

	res, err := svc.Execute(ctx, aCust, from.ID, to.AccountNumber, 400, "xfer-1")
	if err != nil {
		t.Fatal(err)
	}
	if res.From.BalanceCents != 600 || res.To.BalanceCents != 400 {
		t.Fatalf("balances from=%d to=%d", res.From.BalanceCents, res.To.BalanceCents)
	}

	replay, err := svc.Execute(ctx, aCust, from.ID, to.AccountNumber, 400, "xfer-1")
	if err != nil || !replay.Idempotent {
		t.Fatalf("replay err=%v idempotent=%v", err, replay.Idempotent)
	}
	if replay.From.BalanceCents != 600 {
		t.Fatalf("double pay: %d", replay.From.BalanceCents)
	}
}

func TestTransferRejectsNonDigitAccountNumber(t *testing.T) {
	ctx := context.Background()
	store := account.NewMemStore()
	accounts := account.NewService(store)
	svc := NewService(store)
	aCust := uuid.New()
	from, err := accounts.Open(ctx, aCust)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, _ = accounts.Fund(ctx, aCust, from.ID, 100, "fund")
	_, err = svc.Execute(ctx, aCust, from.ID, "TB00000001", 10, "xfer")
	if !errors.Is(err, account.ErrInvalidRequest) {
		t.Fatalf("got %v", err)
	}
}

func TestTransferInsufficientFunds(t *testing.T) {
	ctx := context.Background()
	store := account.NewMemStore()
	accounts := account.NewService(store)
	svc := NewService(store)
	aCust := uuid.New()
	bCust := uuid.New()
	from, _ := accounts.Open(ctx, aCust)
	to, _ := accounts.Open(ctx, bCust)
	_, err := svc.Execute(ctx, aCust, from.ID, to.AccountNumber, 1, "xfer")
	if !errors.Is(err, account.ErrInsufficient) {
		t.Fatalf("got %v", err)
	}
}

func TestTransferFrozen(t *testing.T) {
	ctx := context.Background()
	store := account.NewMemStore()
	accounts := account.NewService(store)
	svc := NewService(store)
	aCust := uuid.New()
	bCust := uuid.New()
	from, _ := accounts.Open(ctx, aCust)
	to, _ := accounts.Open(ctx, bCust)
	_, _, _, _ = accounts.Fund(ctx, aCust, from.ID, 100, "fund")
	if _, err := accounts.Freeze(ctx, aCust, from.ID); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Execute(ctx, aCust, from.ID, to.AccountNumber, 10, "xfer")
	if !errors.Is(err, account.ErrFrozen) {
		t.Fatalf("got %v", err)
	}
}

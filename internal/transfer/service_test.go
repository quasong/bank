package transfer

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"bank/internal/account"
	"bank/internal/ledger"
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

	res, err := svc.Execute(ctx, aCust, from.ID, to.AccountNumber, 400, "xfer-1", "Rent")
	if err != nil {
		t.Fatal(err)
	}
	if res.From.BalanceCents != 600 || res.To.BalanceCents != 400 {
		t.Fatalf("balances from=%d to=%d", res.From.BalanceCents, res.To.BalanceCents)
	}

	replay, err := svc.Execute(ctx, aCust, from.ID, to.AccountNumber, 400, "xfer-1", "ignored")
	if err != nil || !replay.Idempotent {
		t.Fatalf("replay err=%v idempotent=%v", err, replay.Idempotent)
	}
	if replay.From.BalanceCents != 600 {
		t.Fatalf("double pay: %d", replay.From.BalanceCents)
	}
	if replay.Journal.Note != "Rent" {
		t.Fatalf("replay note %q", replay.Journal.Note)
	}

	fromItems, err := accounts.Activity(ctx, aCust, from.ID, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	var xferEntry ledger.Entry
	for _, e := range fromItems {
		if e.Kind == ledger.KindTransfer {
			xferEntry = e
			break
		}
	}
	if xferEntry.CounterpartyNumber != to.AccountNumber || xferEntry.SignedCents != -400 || xferEntry.Note != "Rent" {
		t.Fatalf("from activity %+v", fromItems)
	}
	toItems, err := accounts.Activity(ctx, bCust, to.ID, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(toItems) != 1 || toItems[0].CounterpartyNumber != from.AccountNumber || toItems[0].SignedCents != 400 || toItems[0].Note != "Rent" {
		t.Fatalf("to activity %+v", toItems)
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
	_, err = svc.Execute(ctx, aCust, from.ID, "TB00000001", 10, "xfer", "")
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
	_, err := svc.Execute(ctx, aCust, from.ID, to.AccountNumber, 1, "xfer", "")
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
	_, err := svc.Execute(ctx, aCust, from.ID, to.AccountNumber, 10, "xfer", "")
	if !errors.Is(err, account.ErrFrozen) {
		t.Fatalf("got %v", err)
	}
}

func TestTransferRejectsLongNote(t *testing.T) {
	ctx := context.Background()
	store := account.NewMemStore()
	accounts := account.NewService(store)
	svc := NewService(store)
	aCust := uuid.New()
	bCust := uuid.New()
	from, _ := accounts.Open(ctx, aCust)
	to, _ := accounts.Open(ctx, bCust)
	_, _, _, _ = accounts.Fund(ctx, aCust, from.ID, 100, "fund")
	_, err := svc.Execute(ctx, aCust, from.ID, to.AccountNumber, 10, "xfer", strings.Repeat("n", 41))
	if !errors.Is(err, ledger.ErrInvalidNote) {
		t.Fatalf("got %v", err)
	}
}

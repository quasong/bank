package payee

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"bank/internal/account"
	"bank/internal/currency"
)

func testEnv(t *testing.T) (*Service, *account.Service, *account.MemStore, uuid.UUID, uuid.UUID, account.Account, account.Account) {
	t.Helper()
	ctx := context.Background()
	accounts := account.NewMemStore()
	acctSvc := account.NewService(accounts)
	aCust := uuid.New()
	bCust := uuid.New()
	from, err := acctSvc.Open(ctx, aCust)
	if err != nil {
		t.Fatal(err)
	}
	to, err := acctSvc.Open(ctx, bCust)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(NewMemStore(), accounts)
	return svc, acctSvc, accounts, aCust, bCust, from, to
}

func TestUpsertDefaultsNameToAccountNumber(t *testing.T) {
	ctx := context.Background()
	svc, _, _, aCust, _, _, to := testEnv(t)
	p, err := svc.Upsert(ctx, aCust, to.AccountNumber, "  ")
	if err != nil {
		t.Fatal(err)
	}
	if p.DisplayName != to.AccountNumber || p.AccountNumber != to.AccountNumber {
		t.Fatalf("%+v", p)
	}
}

func TestUpsertKeepsNameUnlessProvided(t *testing.T) {
	ctx := context.Background()
	svc, _, _, aCust, _, _, to := testEnv(t)
	first, err := svc.Upsert(ctx, aCust, to.AccountNumber, "Ada")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond)
	again, err := svc.Upsert(ctx, aCust, to.AccountNumber, "")
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != first.ID {
		t.Fatalf("expected same payee %s vs %s", first.ID, again.ID)
	}
	if again.DisplayName != "Ada" {
		t.Fatalf("name overwritten: %s", again.DisplayName)
	}
	if !again.LastUsedAt.After(first.LastUsedAt) && !again.LastUsedAt.Equal(first.LastUsedAt) {
		t.Fatalf("last used not bumped")
	}
	renamed, err := svc.Upsert(ctx, aCust, to.AccountNumber, "  Bob  ")
	if err != nil || renamed.DisplayName != "Bob" {
		t.Fatalf("%+v %v", renamed, err)
	}
}

func TestUpsertRejectsOwnAccount(t *testing.T) {
	ctx := context.Background()
	svc, _, _, aCust, _, from, _ := testEnv(t)
	_, err := svc.Upsert(ctx, aCust, from.AccountNumber, "Me")
	if !errors.Is(err, ErrOwnAccount) {
		t.Fatalf("got %v", err)
	}
}

func TestUpsertRejectsUnknownNumber(t *testing.T) {
	ctx := context.Background()
	svc, _, _, aCust, _, _, _ := testEnv(t)
	_, err := svc.Upsert(ctx, aCust, currency.RoutingABA+"00000000", "Ghost")
	if !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestUpsertRejectsLongName(t *testing.T) {
	ctx := context.Background()
	svc, _, _, aCust, _, _, to := testEnv(t)
	name := strings.Repeat("n", MaxNameLen+1)
	_, err := svc.Upsert(ctx, aCust, to.AccountNumber, name)
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("got %v", err)
	}
	if utf8.RuneCountInString(strings.Repeat("n", MaxNameLen)) != MaxNameLen {
		t.Fatal("setup")
	}
	if _, err := svc.Upsert(ctx, aCust, to.AccountNumber, strings.Repeat("n", MaxNameLen)); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteAndList(t *testing.T) {
	ctx := context.Background()
	svc, acctSvc, _, aCust, _, _, to := testEnv(t)
	otherCust := uuid.New()
	other, err := acctSvc.Open(ctx, otherCust)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Upsert(ctx, aCust, to.AccountNumber, "Ada"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond)
	second, err := svc.Upsert(ctx, aCust, other.AccountNumber, "Bea")
	if err != nil {
		t.Fatal(err)
	}
	list, err := svc.List(ctx, aCust)
	if err != nil || len(list) != 2 || list[0].DisplayName != "Bea" {
		t.Fatalf("%+v %v", list, err)
	}
	if err := svc.Delete(ctx, aCust, second.ID); err != nil {
		t.Fatal(err)
	}
	list, err = svc.List(ctx, aCust)
	if err != nil || len(list) != 1 || list[0].DisplayName != "Ada" {
		t.Fatalf("%+v %v", list, err)
	}
}

func TestRenameAndResetName(t *testing.T) {
	ctx := context.Background()
	svc, _, _, aCust, _, _, to := testEnv(t)
	p, err := svc.Upsert(ctx, aCust, to.AccountNumber, "Ada")
	if err != nil {
		t.Fatal(err)
	}
	used := p.LastUsedAt
	renamed, err := svc.Rename(ctx, aCust, p.ID, "  Bob  ")
	if err != nil || renamed.DisplayName != "Bob" || renamed.ID != p.ID {
		t.Fatalf("%+v %v", renamed, err)
	}
	if renamed.LastUsedAt != used {
		t.Fatalf("rename should not bump last used")
	}
	reset, err := svc.Rename(ctx, aCust, p.ID, "   ")
	if err != nil || reset.DisplayName != to.AccountNumber {
		t.Fatalf("%+v %v", reset, err)
	}
	_, err = svc.Rename(ctx, aCust, p.ID, strings.Repeat("n", MaxNameLen+1))
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("long name %v", err)
	}
	_, err = svc.Rename(ctx, aCust, uuid.New(), "Ada")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

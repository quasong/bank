package ledger

import (
	"testing"

	"github.com/google/uuid"

	"bank/internal/currency"
)

func TestValidateBalanced(t *testing.T) {
	b := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	if err := Validate([]Line{
		{LedgerAccountID: VaultID, Side: Debit, AmountCents: 100},
		{LedgerAccountID: b, Side: Credit, AmountCents: 100},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestValidateUnbalanced(t *testing.T) {
	b := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	if err := Validate([]Line{
		{LedgerAccountID: VaultID, Side: Debit, AmountCents: 100},
		{LedgerAccountID: b, Side: Credit, AmountCents: 50},
	}); err != ErrUnbalanced {
		t.Fatalf("got %v", err)
	}
}

func TestValidateRejectsNonPositive(t *testing.T) {
	b := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	if err := Validate([]Line{
		{LedgerAccountID: VaultID, Side: Debit, AmountCents: 0},
		{LedgerAccountID: b, Side: Credit, AmountCents: 0},
	}); err != ErrInvalidAmount {
		t.Fatalf("got %v", err)
	}
}

func TestValidateTooFewLines(t *testing.T) {
	if err := Validate([]Line{{LedgerAccountID: VaultID, Side: Debit, AmountCents: 1}}); err != ErrUnbalanced {
		t.Fatalf("got %v", err)
	}
}

func TestValidatePerCurrency(t *testing.T) {
	usd := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	eurVault := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	liabUSD := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	liabEUR := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	if err := Validate([]Line{
		{LedgerAccountID: liabUSD, Side: Debit, AmountCents: 100, Currency: currency.USD},
		{LedgerAccountID: usd, Side: Credit, AmountCents: 100, Currency: currency.USD},
		{LedgerAccountID: eurVault, Side: Debit, AmountCents: 85, Currency: currency.EUR},
		{LedgerAccountID: liabEUR, Side: Credit, AmountCents: 85, Currency: currency.EUR},
	}); err != nil {
		t.Fatal(err)
	}
	if err := Validate([]Line{
		{LedgerAccountID: liabUSD, Side: Debit, AmountCents: 100, Currency: currency.USD},
		{LedgerAccountID: usd, Side: Credit, AmountCents: 90, Currency: currency.USD},
		{LedgerAccountID: eurVault, Side: Debit, AmountCents: 85, Currency: currency.EUR},
		{LedgerAccountID: liabEUR, Side: Credit, AmountCents: 85, Currency: currency.EUR},
	}); err != ErrUnbalanced {
		t.Fatalf("got %v", err)
	}
}

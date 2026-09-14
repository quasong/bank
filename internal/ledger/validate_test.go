package ledger

import (
	"testing"

	"github.com/google/uuid"
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

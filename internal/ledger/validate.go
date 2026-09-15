package ledger

import (
	"errors"

	"github.com/google/uuid"

	"bank/internal/currency"
)

var (
	ErrUnbalanced    = errors.New("journal is not balanced")
	ErrInvalidAmount = errors.New("invalid amount")
	ErrInvalidLine   = errors.New("invalid journal line")
	ErrNotFound      = errors.New("journal not found")
	ErrIdempotency   = errors.New("idempotency key required")
)

func Validate(lines []Line) error {
	if len(lines) < 2 {
		return ErrUnbalanced
	}
	type tot struct{ debit, credit int64 }
	by := map[currency.Code]*tot{}
	for _, ln := range lines {
		if ln.LedgerAccountID == uuid.Nil {
			return ErrInvalidLine
		}
		if ln.AmountCents <= 0 {
			return ErrInvalidAmount
		}
		cc := ln.Currency
		t := by[cc]
		if t == nil {
			t = &tot{}
			by[cc] = t
		}
		switch ln.Side {
		case Debit:
			t.debit += ln.AmountCents
		case Credit:
			t.credit += ln.AmountCents
		default:
			return ErrInvalidLine
		}
	}
	for _, t := range by {
		if t.debit != t.credit {
			return ErrUnbalanced
		}
	}
	return nil
}

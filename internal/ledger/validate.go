package ledger

import (
	"errors"

	"github.com/google/uuid"
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
	var debit, credit int64
	for _, ln := range lines {
		if ln.LedgerAccountID == uuid.Nil {
			return ErrInvalidLine
		}
		if ln.AmountCents <= 0 {
			return ErrInvalidAmount
		}
		switch ln.Side {
		case Debit:
			debit += ln.AmountCents
		case Credit:
			credit += ln.AmountCents
		default:
			return ErrInvalidLine
		}
	}
	if debit != credit {
		return ErrUnbalanced
	}
	return nil
}

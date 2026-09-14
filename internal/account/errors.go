package account

import "errors"

var (
	ErrExists         = errors.New("account exists")
	ErrNotFound       = errors.New("account not found")
	ErrFrozen         = errors.New("account frozen")
	ErrClosed         = errors.New("account closed")
	ErrInsufficient   = errors.New("insufficient funds")
	ErrInvalidAmount  = errors.New("invalid amount")
	ErrIdempotency    = errors.New("idempotency key required")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrInvalidRequest = errors.New("invalid request")
	ErrSameAccount    = errors.New("same account")
	ErrNumberTaken    = errors.New("account number taken")
	ErrHasBalance     = errors.New("account has balance")
)

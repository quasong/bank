package payee

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const MaxNameLen = 40

var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrNotFound       = errors.New("payee not found")
	ErrOwnAccount     = errors.New("cannot save your own account")
)

type Payee struct {
	ID            uuid.UUID
	CustomerID    uuid.UUID
	AccountNumber string
	DisplayName   string
	CreatedAt     time.Time
	LastUsedAt    time.Time
}

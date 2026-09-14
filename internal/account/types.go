package account

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusActive Status = "active"
	StatusFrozen Status = "frozen"
	StatusClosed Status = "closed"
)

func (s Status) OpenForMoney() bool {
	return s == StatusActive
}

type Account struct {
	ID            uuid.UUID
	CustomerID    uuid.UUID
	AccountNumber string
	Status        Status
	BalanceCents  int64
	OpenedAt      time.Time
	LedgerID      uuid.UUID
}

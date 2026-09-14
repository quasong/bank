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

func (s Status) MoneyError() error {
	switch s {
	case StatusActive:
		return nil
	case StatusClosed:
		return ErrClosed
	default:
		return ErrFrozen
	}
}

// Transition returns the status to persist for a requested change.
// Asking for the current status is a no-op. Closed is terminal.
func Transition(acct Account, want Status) (Status, error) {
	if acct.Status == want {
		return acct.Status, nil
	}
	if acct.Status == StatusClosed {
		return "", ErrClosed
	}
	switch want {
	case StatusFrozen:
		if acct.Status == StatusActive {
			return StatusFrozen, nil
		}
	case StatusActive:
		if acct.Status == StatusFrozen {
			return StatusActive, nil
		}
	case StatusClosed:
		if acct.BalanceCents != 0 {
			return "", ErrHasBalance
		}
		if acct.Status == StatusActive || acct.Status == StatusFrozen {
			return StatusClosed, nil
		}
	}
	return "", ErrInvalidRequest
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

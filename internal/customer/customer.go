package customer

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusActive Status = "active"
	StatusLocked Status = "locked"
)

type Customer struct {
	ID        uuid.UUID
	Email     string
	Status    Status
	CreatedAt time.Time
}

func (s Status) Active() bool {
	return s == StatusActive
}

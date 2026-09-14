package ledger

import (
	"time"

	"github.com/google/uuid"
)

// VaultID is the seeded cash (asset) ledger account.
var VaultID = uuid.MustParse("11111111-1111-1111-1111-111111111111")

type Kind string

const (
	KindFunding    Kind = "funding"
	KindTransfer   Kind = "transfer"
	KindWithdrawal Kind = "withdrawal"
)

type Side string

const (
	Debit  Side = "debit"
	Credit Side = "credit"
)

type AccountKind string

const (
	KindAsset     AccountKind = "asset"
	KindLiability AccountKind = "liability"
)

type Line struct {
	LedgerAccountID uuid.UUID
	Side            Side
	AmountCents     int64
}

type Journal struct {
	ID             uuid.UUID
	CreatedAt      time.Time
	Description    string
	Kind           Kind
	IdempotencyKey string
	Lines          []Line
}

type Entry struct {
	JournalID   uuid.UUID
	CreatedAt   time.Time
	Kind        Kind
	Description string
	Side        Side
	AmountCents int64
	SignedCents int64
}

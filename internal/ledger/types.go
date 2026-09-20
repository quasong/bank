package ledger

import (
	"time"

	"github.com/google/uuid"

	"bank/internal/currency"
)

// VaultID is the seeded USD cash (asset) ledger account.
var VaultID = currency.VaultUSD

type Kind string

const (
	KindFunding    Kind = "funding"
	KindTransfer   Kind = "transfer"
	KindWithdrawal Kind = "withdrawal"
	KindFX         Kind = "fx"
	KindMove       Kind = "move"
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
	Currency        currency.Code
}

type Journal struct {
	ID             uuid.UUID
	CreatedAt      time.Time
	Description    string
	Note           string
	Kind           Kind
	IdempotencyKey string
	Lines          []Line
}

type Entry struct {
	JournalID          uuid.UUID
	CreatedAt          time.Time
	Kind               Kind
	Description        string
	Note               string
	Side               Side
	AmountCents        int64
	SignedCents        int64
	CounterpartyNumber string
}

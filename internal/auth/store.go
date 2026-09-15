package auth

import (
	"context"
	"time"

	"github.com/google/uuid"

	"bank/internal/customer"
)

type CustomerRecord struct {
	Customer     customer.Customer
	PasswordHash string
}

type RefreshRecord struct {
	ID         uuid.UUID
	CustomerID uuid.UUID
	TokenHash  string
	ExpiresAt  time.Time
	RevokedAt  *time.Time
}

type NewRefresh struct {
	ID         uuid.UUID
	CustomerID uuid.UUID
	TokenHash  string
	ExpiresAt  time.Time
	IP         string
	UserAgent  string
}

type AuditRecord struct {
	ID        uuid.UUID
	ActorID   *uuid.UUID
	Action    string
	IP        string
	UserAgent string
	Metadata  map[string]string
	CreatedAt time.Time
}

type Store interface {
	CreateCustomer(ctx context.Context, id uuid.UUID, email, passwordHash string, status customer.Status) (customer.Customer, error)
	GetCustomerByEmail(ctx context.Context, email string) (CustomerRecord, error)
	GetCustomerByID(ctx context.Context, id uuid.UUID) (CustomerRecord, error)
	InsertRefreshToken(ctx context.Context, rec NewRefresh) error
	RotateRefreshToken(ctx context.Context, presentedHash string, next NewRefresh, now time.Time) (RefreshRecord, error)
	RevokeRefreshTokenByHash(ctx context.Context, hash string, now time.Time) error
	InsertAudit(ctx context.Context, rec AuditRecord) error
	ListAudit(ctx context.Context, actorID uuid.UUID, limit, offset int32) ([]AuditRecord, error)
}

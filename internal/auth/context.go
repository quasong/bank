package auth

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey int

const customerIDKey ctxKey = 1

func WithCustomerID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, customerIDKey, id)
}

func CustomerIDFrom(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(customerIDKey).(uuid.UUID)
	return id, ok
}

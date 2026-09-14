package auth

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"bank/internal/customer"
)

type memStore struct {
	mu       sync.Mutex
	byID     map[uuid.UUID]CustomerRecord
	byEmail  map[string]CustomerRecord
	tokens   map[string]RefreshRecord
	replaced map[uuid.UUID]uuid.UUID
	audits   []AuditRecord
}

func newMemStore() *memStore {
	return &memStore{
		byID:     make(map[uuid.UUID]CustomerRecord),
		byEmail:  make(map[string]CustomerRecord),
		tokens:   make(map[string]RefreshRecord),
		replaced: make(map[uuid.UUID]uuid.UUID),
	}
}

func (m *memStore) CreateCustomer(_ context.Context, id uuid.UUID, email, passwordHash string, status customer.Status) (customer.Customer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byEmail[email]; ok {
		return customer.Customer{}, ErrEmailTaken
	}
	c := customer.Customer{ID: id, Email: email, Status: status, CreatedAt: time.Now().UTC()}
	rec := CustomerRecord{Customer: c, PasswordHash: passwordHash}
	m.byID[id] = rec
	m.byEmail[email] = rec
	return c, nil
}

func (m *memStore) GetCustomerByEmail(_ context.Context, email string) (CustomerRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.byEmail[email]
	if !ok {
		return CustomerRecord{}, ErrNotFound
	}
	return rec, nil
}

func (m *memStore) GetCustomerByID(_ context.Context, id uuid.UUID) (CustomerRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.byID[id]
	if !ok {
		return CustomerRecord{}, ErrNotFound
	}
	return rec, nil
}

func (m *memStore) InsertRefreshToken(_ context.Context, rec NewRefresh) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[rec.TokenHash] = RefreshRecord{
		ID:         rec.ID,
		CustomerID: rec.CustomerID,
		TokenHash:  rec.TokenHash,
		ExpiresAt:  rec.ExpiresAt,
	}
	return nil
}

func (m *memStore) RotateRefreshToken(_ context.Context, presentedHash string, next NewRefresh, now time.Time) (RefreshRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	old, ok := m.tokens[presentedHash]
	if !ok {
		return RefreshRecord{}, ErrNotFound
	}
	if old.RevokedAt != nil || !old.ExpiresAt.After(now) {
		return old, ErrRefreshInvalid
	}
	revoked := now
	old.RevokedAt = &revoked
	m.tokens[presentedHash] = old
	m.replaced[old.ID] = next.ID
	m.tokens[next.TokenHash] = RefreshRecord{
		ID:         next.ID,
		CustomerID: old.CustomerID,
		TokenHash:  next.TokenHash,
		ExpiresAt:  next.ExpiresAt,
	}
	return old, nil
}

func (m *memStore) RevokeRefreshTokenByHash(_ context.Context, hash string, now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.tokens[hash]
	if !ok || rec.RevokedAt != nil {
		return nil
	}
	rec.RevokedAt = &now
	m.tokens[hash] = rec
	return nil
}

func (m *memStore) InsertAudit(_ context.Context, rec AuditRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.audits = append(m.audits, rec)
	return nil
}

func (m *memStore) lock(email string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.byEmail[email]
	if !ok {
		return
	}
	rec.Customer.Status = customer.StatusLocked
	m.byEmail[email] = rec
	m.byID[rec.Customer.ID] = rec
}

func (m *memStore) actions() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, 0, len(m.audits))
	for _, a := range m.audits {
		out = append(out, a.Action)
	}
	return out
}

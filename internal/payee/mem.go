package payee

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MemStore struct {
	mu     sync.Mutex
	payees map[uuid.UUID]Payee
}

func NewMemStore() *MemStore {
	return &MemStore{payees: make(map[uuid.UUID]Payee)}
}

func (m *MemStore) ListPayees(_ context.Context, customerID uuid.UUID) ([]Payee, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Payee, 0)
	for _, p := range m.payees {
		if p.CustomerID == customerID {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].LastUsedAt.Equal(out[j].LastUsedAt) {
			return out[i].LastUsedAt.After(out[j].LastUsedAt)
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (m *MemStore) UpsertPayee(_ context.Context, customerID uuid.UUID, accountNumber, displayName string, overwriteName bool) (Payee, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	for id, existing := range m.payees {
		if existing.CustomerID == customerID && existing.AccountNumber == accountNumber {
			existing.LastUsedAt = now
			if overwriteName {
				existing.DisplayName = displayName
			}
			m.payees[id] = existing
			return existing, nil
		}
	}
	p := Payee{
		ID:            uuid.New(),
		CustomerID:    customerID,
		AccountNumber: accountNumber,
		DisplayName:   displayName,
		CreatedAt:     now,
		LastUsedAt:    now,
	}
	m.payees[p.ID] = p
	return p, nil
}

func (m *MemStore) DeletePayee(_ context.Context, customerID, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.payees[id]
	if !ok || p.CustomerID != customerID {
		return ErrNotFound
	}
	delete(m.payees, id)
	return nil
}

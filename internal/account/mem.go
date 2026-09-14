package account

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"bank/internal/ledger"
)

type MemStore struct {
	mu       sync.Mutex
	accounts map[uuid.UUID]Account
	journals map[string]ledger.Journal
	activity map[uuid.UUID][]ledger.Entry
}

func NewMemStore() *MemStore {
	return &MemStore{
		accounts: make(map[uuid.UUID]Account),
		journals: make(map[string]ledger.Journal),
		activity: make(map[uuid.UUID][]ledger.Entry),
	}
}

func (m *MemStore) OpenDeposit(_ context.Context, acct Account) (Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.accounts {
		if existing.CustomerID == acct.CustomerID {
			return Account{}, ErrExists
		}
		if existing.AccountNumber == acct.AccountNumber {
			return Account{}, ErrNumberTaken
		}
	}
	acct.OpenedAt = time.Now().UTC()
	acct.BalanceCents = 0
	m.accounts[acct.ID] = acct
	return acct, nil
}

func (m *MemStore) GetByID(_ context.Context, id uuid.UUID) (Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.accounts[id]
	if !ok {
		return Account{}, ErrNotFound
	}
	return a, nil
}

func (m *MemStore) GetByCustomer(_ context.Context, customerID uuid.UUID) (Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, a := range m.accounts {
		if a.CustomerID == customerID {
			return a, nil
		}
	}
	return Account{}, ErrNotFound
}

func (m *MemStore) GetByNumber(_ context.Context, number string) (Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, a := range m.accounts {
		if a.AccountNumber == number {
			return a, nil
		}
	}
	return Account{}, ErrNotFound
}

func (m *MemStore) ListByCustomer(_ context.Context, customerID uuid.UUID) ([]Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Account
	for _, a := range m.accounts {
		if a.CustomerID == customerID {
			out = append(out, a)
		}
	}
	if out == nil {
		out = []Account{}
	}
	return out, nil
}

func (m *MemStore) Freeze(id uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.accounts[id]
	if !ok {
		return
	}
	a.Status = StatusFrozen
	m.accounts[id] = a
}

func (m *MemStore) Post(_ context.Context, journal ledger.Journal, deltas map[uuid.UUID]int64) (ledger.Journal, bool, error) {
	if err := ledger.Validate(journal.Lines); err != nil {
		return ledger.Journal{}, false, err
	}
	if strings.TrimSpace(journal.IdempotencyKey) == "" {
		return ledger.Journal{}, false, ledger.ErrIdempotency
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.journals[journal.IdempotencyKey]; ok {
		return existing, true, nil
	}
	ids := make([]uuid.UUID, 0, len(deltas))
	for id := range deltas {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	for _, id := range ids {
		a, ok := m.accounts[id]
		if !ok {
			return ledger.Journal{}, false, ErrNotFound
		}
		if !a.Status.OpenForMoney() {
			if a.Status == StatusClosed {
				return ledger.Journal{}, false, ErrClosed
			}
			return ledger.Journal{}, false, ErrFrozen
		}
		next := a.BalanceCents + deltas[id]
		if next < 0 {
			return ledger.Journal{}, false, ErrInsufficient
		}
	}
	journal.CreatedAt = time.Now().UTC()
	m.journals[journal.IdempotencyKey] = journal
	for _, id := range ids {
		a := m.accounts[id]
		a.BalanceCents += deltas[id]
		m.accounts[id] = a
	}
	byLedger := make(map[uuid.UUID]uuid.UUID)
	for _, a := range m.accounts {
		byLedger[a.LedgerID] = a.ID
	}
	for _, ln := range journal.Lines {
		acctID, ok := byLedger[ln.LedgerAccountID]
		if !ok {
			continue
		}
		signed := ln.AmountCents
		if ln.Side == ledger.Debit {
			signed = -ln.AmountCents
		}
		m.activity[acctID] = append([]ledger.Entry{{
			JournalID:   journal.ID,
			CreatedAt:   journal.CreatedAt,
			Kind:        journal.Kind,
			Description: journal.Description,
			Side:        ln.Side,
			AmountCents: ln.AmountCents,
			SignedCents: signed,
		}}, m.activity[acctID]...)
	}
	return journal, false, nil
}

func (m *MemStore) Activity(_ context.Context, accountID uuid.UUID, limit, offset int32) ([]ledger.Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	all := m.activity[accountID]
	if offset >= int32(len(all)) {
		return []ledger.Entry{}, nil
	}
	all = all[offset:]
	if limit > 0 && int32(len(all)) > limit {
		all = all[:limit]
	}
	out := make([]ledger.Entry, len(all))
	copy(out, all)
	return out, nil
}

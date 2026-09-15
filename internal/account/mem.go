package account

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"bank/internal/currency"
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
	if acct.Currency == "" {
		acct.Currency = currency.USD
	}
	for _, existing := range m.accounts {
		if existing.CustomerID == acct.CustomerID && existing.Currency == acct.Currency {
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

func (m *MemStore) GetByCustomerCurrency(_ context.Context, customerID uuid.UUID, ccy currency.Code) (Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ccy == "" {
		ccy = currency.USD
	}
	for _, a := range m.accounts {
		if a.CustomerID == customerID && a.Currency == ccy {
			return a, nil
		}
	}
	return Account{}, ErrNotFound
}

func (m *MemStore) GetByNumber(_ context.Context, number string) (Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n, _, ok := currency.Normalize(number)
	if !ok {
		return Account{}, ErrNotFound
	}
	for _, a := range m.accounts {
		if a.AccountNumber == n {
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
	sort.Slice(out, func(i, j int) bool {
		ri, rj := out[i].Currency.Rank(), out[j].Currency.Rank()
		if ri != rj {
			return ri < rj
		}
		return out[i].OpenedAt.Before(out[j].OpenedAt)
	})
	if out == nil {
		out = []Account{}
	}
	return out, nil
}

func (m *MemStore) ListAll(_ context.Context) ([]Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Account, 0, len(m.accounts))
	for _, a := range m.accounts {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CustomerID != out[j].CustomerID {
			return out[i].CustomerID.String() < out[j].CustomerID.String()
		}
		return out[i].OpenedAt.Before(out[j].OpenedAt)
	})
	return out, nil
}

func (m *MemStore) SetNumber(_ context.Context, id uuid.UUID, number string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.accounts[id]
	if !ok {
		return ErrNotFound
	}
	n, _, ok := currency.Normalize(number)
	if !ok {
		return ErrInvalidRequest
	}
	for _, existing := range m.accounts {
		if existing.ID != id && existing.AccountNumber == n {
			return ErrNumberTaken
		}
	}
	a.AccountNumber = n
	m.accounts[id] = a
	return nil
}

func (m *MemStore) SetStatus(_ context.Context, id uuid.UUID, next Status) (Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.accounts[id]
	if !ok {
		return Account{}, ErrNotFound
	}
	want, err := Transition(a, next)
	if err != nil {
		return Account{}, err
	}
	a.Status = want
	m.accounts[id] = a
	return a, nil
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
		if err := a.Status.MoneyError(); err != nil {
			return ledger.Journal{}, false, err
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
	journalAccounts := make([]uuid.UUID, 0, 2)
	seen := make(map[uuid.UUID]bool)
	for _, ln := range journal.Lines {
		acctID, ok := byLedger[ln.LedgerAccountID]
		if !ok || seen[acctID] {
			continue
		}
		seen[acctID] = true
		journalAccounts = append(journalAccounts, acctID)
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
		counterparty := ""
		for _, otherID := range journalAccounts {
			if otherID != acctID {
				counterparty = m.accounts[otherID].AccountNumber
				break
			}
		}
		m.activity[acctID] = append([]ledger.Entry{{
			JournalID:          journal.ID,
			CreatedAt:          journal.CreatedAt,
			Kind:               journal.Kind,
			Description:        journal.Description,
			Note:               journal.Note,
			Side:               ln.Side,
			AmountCents:        ln.AmountCents,
			SignedCents:        signed,
			CounterpartyNumber: counterparty,
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

package account

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"bank/internal/audit"
	"bank/internal/auth"
)

type memAudit struct {
	recs []auth.AuditRecord
}

func (m *memAudit) InsertAudit(_ context.Context, rec auth.AuditRecord) error {
	m.recs = append(m.recs, rec)
	return nil
}

func TestFundRejectsFloatCents(t *testing.T) {
	store := NewMemStore()
	svc := NewService(store)
	h := NewHandler(svc, nil, nil)
	cid := uuid.New()
	acct, err := svc.Open(t.Context(), cid)
	if err != nil {
		t.Fatal(err)
	}

	r := chi.NewRouter()
	r.Post("/api/v1/accounts/{id}/funding", func(w http.ResponseWriter, req *http.Request) {
		h.Fund(w, req.WithContext(auth.WithCustomerID(req.Context(), cid)))
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts/"+acct.ID.String()+"/funding", strings.NewReader(`{"amount_cents":10.5,"idempotency_key":"k"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "invalid_request") {
		t.Fatalf("body %s", rec.Body.String())
	}
}

func TestWithdrawRejectsFloatCents(t *testing.T) {
	store := NewMemStore()
	svc := NewService(store)
	h := NewHandler(svc, nil, nil)
	cid := uuid.New()
	acct, err := svc.Open(t.Context(), cid)
	if err != nil {
		t.Fatal(err)
	}

	r := chi.NewRouter()
	r.Post("/api/v1/accounts/{id}/withdrawals", func(w http.ResponseWriter, req *http.Request) {
		h.Withdraw(w, req.WithContext(auth.WithCustomerID(req.Context(), cid)))
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts/"+acct.ID.String()+"/withdrawals", strings.NewReader(`{"amount_cents":10.5,"idempotency_key":"k"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}

func TestFundWritesAuditOnce(t *testing.T) {
	store := NewMemStore()
	svc := NewService(store)
	log := &memAudit{}
	h := NewHandler(svc, nil, log)
	cid := uuid.New()
	acct, err := svc.Open(t.Context(), cid)
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	r.Post("/api/v1/accounts/{id}/funding", func(w http.ResponseWriter, req *http.Request) {
		h.Fund(w, req.WithContext(auth.WithCustomerID(req.Context(), cid)))
	})
	body := `{"amount_cents":100,"idempotency_key":"in-1"}`
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts/"+acct.ID.String()+"/funding", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
			t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
		}
	}
	if len(log.recs) != 1 || log.recs[0].Action != audit.Funding || log.recs[0].Metadata["amount_cents"] != "100" {
		t.Fatalf("%+v", log.recs)
	}
}

func TestFreezeWritesAuditOnce(t *testing.T) {
	store := NewMemStore()
	svc := NewService(store)
	log := &memAudit{}
	h := NewHandler(svc, nil, log)
	cid := uuid.New()
	acct, err := svc.Open(t.Context(), cid)
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	r.Post("/api/v1/accounts/{id}/freeze", func(w http.ResponseWriter, req *http.Request) {
		h.Freeze(w, req.WithContext(auth.WithCustomerID(req.Context(), cid)))
	})
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts/"+acct.ID.String()+"/freeze", nil)
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
		}
	}
	if len(log.recs) != 1 || log.recs[0].Action != audit.AccountFreeze {
		t.Fatalf("%+v", log.recs)
	}
}

package account

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"bank/internal/auth"
)

func TestFundRejectsFloatCents(t *testing.T) {
	store := NewMemStore()
	svc := NewService(store)
	h := NewHandler(svc)
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

package payee

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"bank/internal/account"
	"bank/internal/auth"
)

func TestCreateRejectsOwnAccount(t *testing.T) {
	ctx := t.Context()
	accounts := account.NewMemStore()
	acctSvc := account.NewService(accounts)
	cid := uuid.New()
	mine, err := acctSvc.Open(ctx, cid)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(NewService(NewMemStore(), accounts))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payees", strings.NewReader(`{"account_number":"`+mine.AccountNumber+`","display_name":"Me"}`))
	req.Header.Set("Content-Type", "application/json")
	h.Create(rec, req.WithContext(auth.WithCustomerID(req.Context(), cid)))
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "own_account") {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}

func TestCreateAndList(t *testing.T) {
	ctx := t.Context()
	accounts := account.NewMemStore()
	acctSvc := account.NewService(accounts)
	aCust := uuid.New()
	bCust := uuid.New()
	if _, err := acctSvc.Open(ctx, aCust); err != nil {
		t.Fatal(err)
	}
	to, err := acctSvc.Open(ctx, bCust)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(NewService(NewMemStore(), accounts))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payees", strings.NewReader(`{"account_number":"`+to.AccountNumber+`","display_name":"Ada"}`))
	req.Header.Set("Content-Type", "application/json")
	h.Create(rec, req.WithContext(auth.WithCustomerID(req.Context(), aCust)))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Ada") {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/payees", nil)
	h.List(listRec, listReq.WithContext(auth.WithCustomerID(listReq.Context(), aCust)))
	if listRec.Code != http.StatusOK || !strings.Contains(listRec.Body.String(), to.AccountNumber) {
		t.Fatalf("status %d body %s", listRec.Code, listRec.Body.String())
	}
}

func TestRenameAndDeleteHTTP(t *testing.T) {
	ctx := t.Context()
	accounts := account.NewMemStore()
	acctSvc := account.NewService(accounts)
	aCust := uuid.New()
	bCust := uuid.New()
	if _, err := acctSvc.Open(ctx, aCust); err != nil {
		t.Fatal(err)
	}
	to, err := acctSvc.Open(ctx, bCust)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(NewMemStore(), accounts)
	h := NewHandler(svc)
	created, err := svc.Upsert(ctx, aCust, to.AccountNumber, "Ada")
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	r.Patch("/api/v1/payees/{id}", func(w http.ResponseWriter, req *http.Request) {
		h.Rename(w, req.WithContext(auth.WithCustomerID(req.Context(), aCust)))
	})
	r.Delete("/api/v1/payees/{id}", func(w http.ResponseWriter, req *http.Request) {
		h.Delete(w, req.WithContext(auth.WithCustomerID(req.Context(), aCust)))
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/payees/"+created.ID.String(), strings.NewReader(`{"display_name":"  Bea  "}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"display_name":"Bea"`) {
		t.Fatalf("rename status %d body %s", rec.Code, rec.Body.String())
	}

	del := httptest.NewRecorder()
	r.ServeHTTP(del, httptest.NewRequest(http.MethodDelete, "/api/v1/payees/"+created.ID.String(), nil))
	if del.Code != http.StatusNoContent {
		t.Fatalf("delete status %d body %s", del.Code, del.Body.String())
	}

	missing := httptest.NewRecorder()
	missReq := httptest.NewRequest(http.MethodPatch, "/api/v1/payees/"+created.ID.String(), strings.NewReader(`{"display_name":"Ada"}`))
	missReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(missing, missReq)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing status %d body %s", missing.Code, missing.Body.String())
	}
}

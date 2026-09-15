package transfer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"bank/internal/account"
	"bank/internal/audit"
	"bank/internal/auth"
	"bank/internal/payee"
)

type memAudit struct {
	recs []auth.AuditRecord
}

func (m *memAudit) InsertAudit(_ context.Context, rec auth.AuditRecord) error {
	m.recs = append(m.recs, rec)
	return nil
}

func TestCreateRemembersPayee(t *testing.T) {
	ctx := t.Context()
	accounts := account.NewMemStore()
	acctSvc := account.NewService(accounts)
	aCust := uuid.New()
	bCust := uuid.New()
	from, err := acctSvc.Open(ctx, aCust)
	if err != nil {
		t.Fatal(err)
	}
	to, err := acctSvc.Open(ctx, bCust)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := acctSvc.Fund(ctx, aCust, from.ID, 1000, "in"); err != nil {
		t.Fatal(err)
	}
	payees := payee.NewMemStore()
	payeeSvc := payee.NewService(payees, accounts)
	h := NewHandler(NewService(accounts), payeeSvc, nil)

	body := `{"from_account_id":"` + from.ID.String() + `","to_account_number":"` + to.AccountNumber + `","amount_cents":250,"idempotency_key":"k1","payee_name":"Ada"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.Create(rec, req.WithContext(auth.WithCustomerID(req.Context(), aCust)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Transfer struct {
			Receipt          string `json:"receipt"`
			ToAccountNumber  string `json:"to_account_number"`
			FromBalanceCents int64  `json:"from_balance_cents"`
		} `json:"transfer"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Transfer.Receipt == "" || payload.Transfer.ToAccountNumber != to.AccountNumber || payload.Transfer.FromBalanceCents != 750 {
		t.Fatalf("%+v", payload)
	}
	list, err := payeeSvc.List(ctx, aCust)
	if err != nil || len(list) != 1 || list[0].DisplayName != "Ada" || list[0].AccountNumber != to.AccountNumber {
		t.Fatalf("%+v %v", list, err)
	}
}

func TestCreateStoresNoteAndAudit(t *testing.T) {
	ctx := t.Context()
	accounts := account.NewMemStore()
	acctSvc := account.NewService(accounts)
	aCust := uuid.New()
	bCust := uuid.New()
	from, err := acctSvc.Open(ctx, aCust)
	if err != nil {
		t.Fatal(err)
	}
	to, err := acctSvc.Open(ctx, bCust)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := acctSvc.Fund(ctx, aCust, from.ID, 1000, "in"); err != nil {
		t.Fatal(err)
	}
	log := &memAudit{}
	h := NewHandler(NewService(accounts), nil, log)
	body := `{"from_account_id":"` + from.ID.String() + `","to_account_number":"` + to.AccountNumber + `","amount_cents":100,"idempotency_key":"k-note","note":"  Rent  "}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.Create(rec, req.WithContext(auth.WithCustomerID(req.Context(), aCust)))
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"note":"Rent"`) {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	if len(log.recs) != 1 || log.recs[0].Action != audit.Transfer || log.recs[0].Metadata["amount_cents"] != "100" || log.recs[0].Metadata["note"] != "Rent" {
		t.Fatalf("%+v", log.recs)
	}
	replay := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	h.Create(replay, req2.WithContext(auth.WithCustomerID(req2.Context(), aCust)))
	if replay.Code != http.StatusOK || len(log.recs) != 1 {
		t.Fatalf("replay status %d audits %d", replay.Code, len(log.recs))
	}
}

package transfer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"bank/internal/account"
	"bank/internal/audit"
	"bank/internal/auth"
	"bank/internal/ledger"
	"bank/internal/moneyjson"
	"bank/internal/payee"
)

type Handler struct {
	svc    *Service
	payees PayeeBook
	audit  Auditor
}

type PayeeBook interface {
	Upsert(ctx context.Context, customerID uuid.UUID, accountNumber, displayName string) (payee.Payee, error)
}

type Auditor interface {
	InsertAudit(ctx context.Context, rec auth.AuditRecord) error
}

func NewHandler(svc *Service, payees PayeeBook, auditor Auditor) *Handler {
	return &Handler{svc: svc, payees: payees, audit: auditor}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	customerID, ok := auth.CustomerIDFrom(r.Context())
	if !ok {
		auth.WriteError(w, http.StatusUnauthorized, "unauthorized", "please sign in again")
		return
	}
	var raw map[string]json.RawMessage
	if err := moneyjson.Decode(r.Body, &raw); err != nil {
		auth.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	amount, err := moneyjson.PositiveCents(raw["amount_cents"])
	if err != nil {
		auth.WriteError(w, http.StatusBadRequest, "invalid_request", "amount_cents must be a positive integer")
		return
	}
	fromStr, err := moneyjson.String(raw["from_account_id"])
	if err != nil {
		auth.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	fromID, err := uuid.Parse(fromStr)
	if err != nil {
		auth.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	toNumber, err := moneyjson.String(raw["to_account_number"])
	if err != nil {
		auth.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	key, _ := moneyjson.String(raw["idempotency_key"])
	payeeName, _ := moneyjson.String(raw["payee_name"])
	note, _ := moneyjson.String(raw["note"])
	res, err := h.svc.Execute(r.Context(), customerID, fromID, toNumber, amount, key, note)
	if err != nil {
		if errors.Is(err, ledger.ErrInvalidNote) {
			auth.WriteError(w, http.StatusBadRequest, "invalid_request", "note must be at most 40 characters")
			return
		}
		account.WriteHTTPError(w, err)
		return
	}
	if h.payees != nil {
		_, _ = h.payees.Upsert(r.Context(), customerID, res.To.AccountNumber, payeeName)
	}
	if h.audit != nil && !res.Idempotent {
		meta := map[string]string{
			"account_id":        res.From.ID.String(),
			"account_number":    res.From.AccountNumber,
			"to_account_number": res.To.AccountNumber,
			"amount_cents":      strconv.FormatInt(res.AmountCents, 10),
			"journal_id":        res.Journal.ID.String(),
		}
		if res.Journal.Note != "" {
			meta["note"] = res.Journal.Note
		}
		_ = h.audit.InsertAudit(r.Context(), auth.AuditRecord{
			ID:        uuid.New(),
			ActorID:   &customerID,
			Action:    audit.Transfer,
			IP:        audit.ClientIP(r),
			UserAgent: r.UserAgent(),
			Metadata:  meta,
		})
	}
	status := http.StatusCreated
	if res.Idempotent {
		status = http.StatusOK
	}
	created := ""
	if !res.Journal.CreatedAt.IsZero() {
		created = res.Journal.CreatedAt.UTC().Format(time.RFC3339)
	}
	body := map[string]any{
		"journal_id":          res.Journal.ID.String(),
		"receipt":             ledger.ReceiptCode(res.Journal.ID),
		"kind":                string(res.Journal.Kind),
		"description":         res.Journal.Description,
		"amount_cents":        res.AmountCents,
		"from_account_id":     res.From.ID.String(),
		"from_account_number": res.From.AccountNumber,
		"from_balance_cents":  res.From.BalanceCents,
		"to_account_id":       res.To.ID.String(),
		"to_account_number":   res.To.AccountNumber,
		"created_at":          created,
		"replay":              res.Idempotent,
	}
	if res.Journal.Note != "" {
		body["note"] = res.Journal.Note
	}
	writeJSON(w, status, map[string]any{"transfer": body})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

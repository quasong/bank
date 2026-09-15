package fx

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"bank/internal/account"
	"bank/internal/audit"
	"bank/internal/auth"
	"bank/internal/currency"
	"bank/internal/ledger"
	"bank/internal/moneyjson"
)

type Handler struct {
	svc   *Service
	audit Auditor
}

type Auditor interface {
	InsertAudit(ctx context.Context, rec auth.AuditRecord) error
}

func NewHandler(svc *Service, auditor Auditor) *Handler {
	return &Handler{svc: svc, audit: auditor}
}

func (h *Handler) Quote(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.CustomerIDFrom(r.Context()); !ok {
		auth.WriteError(w, http.StatusUnauthorized, "unauthorized", "please sign in again")
		return
	}
	from, ok := currency.Parse(r.URL.Query().Get("from"))
	if !ok {
		auth.WriteError(w, http.StatusBadRequest, "invalid_request", "unsupported currency")
		return
	}
	to, ok := currency.Parse(r.URL.Query().Get("to"))
	if !ok {
		auth.WriteError(w, http.StatusBadRequest, "invalid_request", "unsupported currency")
		return
	}
	var amount int64
	if v := r.URL.Query().Get("amount_cents"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 0 {
			auth.WriteError(w, http.StatusBadRequest, "invalid_request", "amount_cents must be a positive integer")
			return
		}
		amount = n
	}
	q, err := h.svc.Quote(r.Context(), from, to, amount)
	if err != nil {
		account.WriteHTTPError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"quote": quoteBody(q)})
}

func (h *Handler) Convert(w http.ResponseWriter, r *http.Request) {
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
	toStr, err := moneyjson.String(raw["to_currency"])
	if err != nil {
		auth.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	toCCY, ok := currency.Parse(toStr)
	if !ok {
		auth.WriteError(w, http.StatusBadRequest, "invalid_request", "unsupported currency")
		return
	}
	key, _ := moneyjson.String(raw["idempotency_key"])
	res, err := h.svc.Convert(r.Context(), customerID, fromID, toCCY, amount, key)
	if err != nil {
		account.WriteHTTPError(w, err)
		return
	}
	if h.audit != nil && !res.Idempotent {
		_ = h.audit.InsertAudit(r.Context(), auth.AuditRecord{
			ID:        uuid.New(),
			ActorID:   &customerID,
			Action:    audit.FX,
			IP:        audit.ClientIP(r),
			UserAgent: r.UserAgent(),
			Metadata: map[string]string{
				"account_id":     res.From.ID.String(),
				"account_number": res.From.AccountNumber,
				"from_currency":  string(res.From.Currency),
				"to_account_id":  res.To.ID.String(),
				"to_currency":    string(res.To.Currency),
				"amount_cents":   strconv.FormatInt(res.FromCents, 10),
				"quote_cents":    strconv.FormatInt(res.ToCents, 10),
				"rate":           res.Rate,
				"journal_id":     res.Journal.ID.String(),
			},
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
	writeJSON(w, status, map[string]any{
		"conversion": map[string]any{
			"journal_id":          res.Journal.ID.String(),
			"receipt":             ledger.ReceiptCode(res.Journal.ID),
			"from_account_id":     res.From.ID.String(),
			"from_account_number": res.From.AccountNumber,
			"from_currency":       string(res.From.Currency),
			"from_balance_cents":  res.From.BalanceCents,
			"to_account_id":       res.To.ID.String(),
			"to_account_number":   res.To.AccountNumber,
			"to_currency":         string(res.To.Currency),
			"to_balance_cents":    res.To.BalanceCents,
			"amount_cents":        res.FromCents,
			"quote_cents":         res.ToCents,
			"rate":                res.Rate,
			"as_of":               res.AsOf,
			"created_at":          created,
			"replay":              res.Idempotent,
		},
	})
}

func quoteBody(q Quote) map[string]any {
	body := map[string]any{
		"from":    string(q.From),
		"to":      string(q.To),
		"rate":    q.Rate,
		"rate_e8": q.RateE8,
		"as_of":   q.AsOf,
	}
	if q.AmountCents > 0 {
		body["amount_cents"] = q.AmountCents
		body["quote_cents"] = q.QuoteCents
	}
	return body
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

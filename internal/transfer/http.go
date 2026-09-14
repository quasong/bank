package transfer

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"bank/internal/account"
	"bank/internal/auth"
	"bank/internal/moneyjson"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
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
	res, err := h.svc.Execute(r.Context(), customerID, fromID, toNumber, amount, key)
	if err != nil {
		account.WriteHTTPError(w, err)
		return
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
		"transfer": map[string]any{
			"journal_id":          res.Journal.ID.String(),
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
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

package account

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"bank/internal/auth"
	"bank/internal/ledger"
	"bank/internal/moneyjson"
)

type Handler struct {
	svc   *Service
	names PayeeNames
}

// PayeeNames attaches saved payee display names onto activity items.
type PayeeNames interface {
	Names(ctx context.Context, customerID uuid.UUID) (map[string]string, error)
}

func NewHandler(svc *Service, names PayeeNames) *Handler {
	return &Handler{svc: svc, names: names}
}

type accountBody struct {
	ID            string `json:"id"`
	AccountNumber string `json:"account_number"`
	Status        string `json:"status"`
	BalanceCents  int64  `json:"balance_cents"`
	OpenedAt      string `json:"opened_at"`
}

type journalBody struct {
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	Description    string `json:"description"`
	IdempotencyKey string `json:"idempotency_key"`
	CreatedAt      string `json:"created_at"`
}

func (h *Handler) Open(w http.ResponseWriter, r *http.Request) {
	customerID, ok := auth.CustomerIDFrom(r.Context())
	if !ok {
		auth.WriteError(w, http.StatusUnauthorized, "unauthorized", "please sign in again")
		return
	}
	acct, err := h.svc.Open(r.Context(), customerID)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"account": toAccountBody(acct)})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	customerID, ok := auth.CustomerIDFrom(r.Context())
	if !ok {
		auth.WriteError(w, http.StatusUnauthorized, "unauthorized", "please sign in again")
		return
	}
	list, err := h.svc.List(r.Context(), customerID)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	bodies := make([]accountBody, 0, len(list))
	for _, a := range list {
		bodies = append(bodies, toAccountBody(a))
	}
	writeJSON(w, http.StatusOK, map[string]any{"accounts": bodies})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	customerID, acctID, ok := pathAccount(w, r)
	if !ok {
		return
	}
	acct, err := h.svc.GetOwned(r.Context(), customerID, acctID)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"account": toAccountBody(acct)})
}

func (h *Handler) Fund(w http.ResponseWriter, r *http.Request) {
	customerID, acctID, ok := pathAccount(w, r)
	if !ok {
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
	key, _ := moneyjson.String(raw["idempotency_key"])
	acct, journal, replay, err := h.svc.Fund(r.Context(), customerID, acctID, amount, key)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	status := http.StatusCreated
	if replay {
		status = http.StatusOK
	}
	writeJSON(w, status, map[string]any{
		"account": toAccountBody(acct),
		"journal": toJournalBody(journal),
		"replay":  replay,
	})
}

func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	customerID, acctID, ok := pathAccount(w, r)
	if !ok {
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
	key, _ := moneyjson.String(raw["idempotency_key"])
	acct, journal, replay, err := h.svc.Withdraw(r.Context(), customerID, acctID, amount, key)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	status := http.StatusCreated
	if replay {
		status = http.StatusOK
	}
	writeJSON(w, status, map[string]any{
		"account": toAccountBody(acct),
		"journal": toJournalBody(journal),
		"replay":  replay,
	})
}

func (h *Handler) Freeze(w http.ResponseWriter, r *http.Request) {
	h.writeStatusChange(w, r, h.svc.Freeze)
}

func (h *Handler) Unfreeze(w http.ResponseWriter, r *http.Request) {
	h.writeStatusChange(w, r, h.svc.Unfreeze)
}

func (h *Handler) Close(w http.ResponseWriter, r *http.Request) {
	h.writeStatusChange(w, r, h.svc.Close)
}

func (h *Handler) writeStatusChange(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context, customerID, accountID uuid.UUID) (Account, error)) {
	customerID, acctID, ok := pathAccount(w, r)
	if !ok {
		return
	}
	acct, err := fn(r.Context(), customerID, acctID)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"account": toAccountBody(acct)})
}

func (h *Handler) Activity(w http.ResponseWriter, r *http.Request) {
	customerID, acctID, ok := pathAccount(w, r)
	if !ok {
		return
	}
	limit := int32(50)
	offset := int32(0)
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			limit = int32(n)
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			offset = int32(n)
		}
	}
	items, err := h.svc.Activity(r.Context(), customerID, acctID, limit, offset)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	var names map[string]string
	if h.names != nil {
		names, _ = h.names.Names(r.Context(), customerID)
	}
	out := make([]map[string]any, 0, len(items))
	for _, e := range items {
		item := map[string]any{
			"journal_id":   e.JournalID.String(),
			"receipt":      ledger.ReceiptCode(e.JournalID),
			"created_at":   e.CreatedAt.UTC().Format(time.RFC3339),
			"kind":         string(e.Kind),
			"description":  e.Description,
			"side":         string(e.Side),
			"amount_cents": e.AmountCents,
			"signed_cents": e.SignedCents,
		}
		if e.CounterpartyNumber != "" {
			item["counterparty_account_number"] = e.CounterpartyNumber
			if name := names[e.CounterpartyNumber]; name != "" {
				item["counterparty_name"] = name
			}
		}
		out = append(out, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func pathAccount(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	customerID, ok := auth.CustomerIDFrom(r.Context())
	if !ok {
		auth.WriteError(w, http.StatusUnauthorized, "unauthorized", "please sign in again")
		return uuid.Nil, uuid.Nil, false
	}
	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		auth.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid account id")
		return uuid.Nil, uuid.Nil, false
	}
	return customerID, id, true
}

func toAccountBody(a Account) accountBody {
	return accountBody{
		ID:            a.ID.String(),
		AccountNumber: a.AccountNumber,
		Status:        string(a.Status),
		BalanceCents:  a.BalanceCents,
		OpenedAt:      a.OpenedAt.UTC().Format(time.RFC3339),
	}
}

func toJournalBody(j ledger.Journal) journalBody {
	created := ""
	if !j.CreatedAt.IsZero() {
		created = j.CreatedAt.UTC().Format(time.RFC3339)
	}
	return journalBody{
		ID:             j.ID.String(),
		Kind:           string(j.Kind),
		Description:    j.Description,
		IdempotencyKey: j.IdempotencyKey,
		CreatedAt:      created,
	}
}

func writeAccountError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidRequest), errors.Is(err, ErrInvalidAmount), errors.Is(err, ErrIdempotency), errors.Is(err, ledger.ErrInvalidAmount), errors.Is(err, ledger.ErrUnbalanced), errors.Is(err, ledger.ErrIdempotency):
		auth.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
	case errors.Is(err, ErrExists):
		auth.WriteError(w, http.StatusConflict, "account_exists", "a deposit account already exists")
	case errors.Is(err, ErrNotFound):
		auth.WriteError(w, http.StatusNotFound, "account_not_found", "account not found")
	case errors.Is(err, ErrFrozen):
		auth.WriteError(w, http.StatusForbidden, "account_frozen", "account cannot move money")
	case errors.Is(err, ErrClosed):
		auth.WriteError(w, http.StatusForbidden, "account_closed", "account is closed")
	case errors.Is(err, ErrHasBalance):
		auth.WriteError(w, http.StatusConflict, "account_has_balance", "withdraw remaining funds before closing")
	case errors.Is(err, ErrInsufficient):
		auth.WriteError(w, http.StatusConflict, "insufficient_funds", "insufficient funds")
	case errors.Is(err, ErrSameAccount):
		auth.WriteError(w, http.StatusBadRequest, "invalid_request", "cannot transfer to the same account")
	case errors.Is(err, ErrUnauthorized):
		auth.WriteError(w, http.StatusUnauthorized, "unauthorized", "please sign in again")
	default:
		auth.WriteError(w, http.StatusInternalServerError, "internal", "service unavailable")
	}
}

func WriteHTTPError(w http.ResponseWriter, err error) {
	writeAccountError(w, err)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

package payee

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"bank/internal/account"
	"bank/internal/auth"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type payeeBody struct {
	ID            string `json:"id"`
	AccountNumber string `json:"account_number"`
	DisplayName   string `json:"display_name"`
	CreatedAt     string `json:"created_at"`
	LastUsedAt    string `json:"last_used_at"`
}

type createBody struct {
	AccountNumber string `json:"account_number"`
	DisplayName   string `json:"display_name"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	customerID, ok := auth.CustomerIDFrom(r.Context())
	if !ok {
		auth.WriteError(w, http.StatusUnauthorized, "unauthorized", "please sign in again")
		return
	}
	items, err := h.svc.List(r.Context(), customerID)
	if err != nil {
		writePayeeError(w, err)
		return
	}
	out := make([]payeeBody, 0, len(items))
	for _, p := range items {
		out = append(out, toPayeeBody(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"payees": out})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	customerID, ok := auth.CustomerIDFrom(r.Context())
	if !ok {
		auth.WriteError(w, http.StatusUnauthorized, "unauthorized", "please sign in again")
		return
	}
	var body createBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		auth.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	p, err := h.svc.Upsert(r.Context(), customerID, body.AccountNumber, body.DisplayName)
	if err != nil {
		writePayeeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"payee": toPayeeBody(p)})
}

type patchBody struct {
	DisplayName string `json:"display_name"`
}

func (h *Handler) Rename(w http.ResponseWriter, r *http.Request) {
	customerID, ok := auth.CustomerIDFrom(r.Context())
	if !ok {
		auth.WriteError(w, http.StatusUnauthorized, "unauthorized", "please sign in again")
		return
	}
	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		auth.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid payee id")
		return
	}
	var body patchBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		auth.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	p, err := h.svc.Rename(r.Context(), customerID, id, body.DisplayName)
	if err != nil {
		writePayeeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"payee": toPayeeBody(p)})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	customerID, ok := auth.CustomerIDFrom(r.Context())
	if !ok {
		auth.WriteError(w, http.StatusUnauthorized, "unauthorized", "please sign in again")
		return
	}
	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		auth.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid payee id")
		return
	}
	if err := h.svc.Delete(r.Context(), customerID, id); err != nil {
		writePayeeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func toPayeeBody(p Payee) payeeBody {
	return payeeBody{
		ID:            p.ID.String(),
		AccountNumber: p.AccountNumber,
		DisplayName:   p.DisplayName,
		CreatedAt:     p.CreatedAt.UTC().Format(time.RFC3339),
		LastUsedAt:    p.LastUsedAt.UTC().Format(time.RFC3339),
	}
}

func writePayeeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidRequest), errors.Is(err, account.ErrInvalidRequest):
		auth.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
	case errors.Is(err, ErrOwnAccount):
		auth.WriteError(w, http.StatusBadRequest, "own_account", "cannot save your own account")
	case errors.Is(err, ErrNotFound):
		auth.WriteError(w, http.StatusNotFound, "payee_not_found", "payee not found")
	case errors.Is(err, account.ErrNotFound):
		auth.WriteError(w, http.StatusNotFound, "account_not_found", "account not found")
	default:
		auth.WriteError(w, http.StatusInternalServerError, "internal", "service unavailable")
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

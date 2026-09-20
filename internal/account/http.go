package account

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"bank/internal/audit"
	"bank/internal/auth"
	"bank/internal/currency"
	"bank/internal/ledger"
	"bank/internal/moneyjson"
)

type Handler struct {
	svc   *Service
	names PayeeNames
	audit Auditor
}

// PayeeNames attaches saved payee display names onto activity items.
type PayeeNames interface {
	Names(ctx context.Context, customerID uuid.UUID) (map[string]string, error)
}

type Auditor interface {
	InsertAudit(ctx context.Context, rec auth.AuditRecord) error
}

func NewHandler(svc *Service, names PayeeNames, auditor Auditor) *Handler {
	return &Handler{svc: svc, names: names, audit: auditor}
}

type accountBody struct {
	ID                     string            `json:"id"`
	Currency               string            `json:"currency"`
	AccountNumber          string            `json:"account_number"`
	AccountNumberFormatted string            `json:"account_number_formatted"`
	Product                string            `json:"product"`
	Label                  string            `json:"label,omitempty"`
	Status                 string            `json:"status"`
	BalanceCents           int64             `json:"balance_cents"`
	OpenedAt               string            `json:"opened_at"`
	Details                map[string]string `json:"details"`
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
	var codes []currency.Code
	raw := map[string]json.RawMessage{}
	if err := moneyjson.Decode(r.Body, &raw); err != nil && !errors.Is(err, io.EOF) {
		auth.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	if s, err := moneyjson.String(raw["currency"]); err == nil && strings.TrimSpace(s) != "" {
		ccy, ok := currency.Parse(s)
		if !ok {
			auth.WriteError(w, http.StatusBadRequest, "invalid_request", "unsupported currency")
			return
		}
		codes = []currency.Code{ccy}
	}
	productRaw, _ := moneyjson.String(raw["product"])
	product, ok := ParseProduct(productRaw)
	if !ok {
		auth.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	var acct Account
	var err error
	if product == ProductJar {
		ccy := currency.USD
		if len(codes) > 0 {
			ccy = codes[0]
		}
		label, _ := moneyjson.String(raw["label"])
		acct, err = h.svc.OpenJar(r.Context(), customerID, ccy, label)
	} else {
		acct, err = h.svc.Open(r.Context(), customerID, codes...)
	}
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
	if !replay {
		h.recordMoney(r, customerID, audit.Funding, acct, journal, amount)
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
	if !replay {
		h.recordMoney(r, customerID, audit.Withdrawal, acct, journal, amount)
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
	h.writeStatusChange(w, r, h.svc.Freeze, audit.AccountFreeze)
}

func (h *Handler) Unfreeze(w http.ResponseWriter, r *http.Request) {
	h.writeStatusChange(w, r, h.svc.Unfreeze, audit.AccountUnfreeze)
}

func (h *Handler) Close(w http.ResponseWriter, r *http.Request) {
	h.writeStatusChange(w, r, h.svc.Close, audit.AccountClose)
}

func (h *Handler) Rename(w http.ResponseWriter, r *http.Request) {
	customerID, acctID, ok := pathAccount(w, r)
	if !ok {
		return
	}
	var raw map[string]json.RawMessage
	if err := moneyjson.Decode(r.Body, &raw); err != nil {
		auth.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	label, _ := moneyjson.String(raw["label"])
	acct, err := h.svc.RenameJar(r.Context(), customerID, acctID, label)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"account": toAccountBody(acct)})
}

func (h *Handler) Move(w http.ResponseWriter, r *http.Request) {
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
	toStr, err := moneyjson.String(raw["to_account_id"])
	if err != nil {
		auth.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	fromID, err := uuid.Parse(fromStr)
	if err != nil {
		auth.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	toID, err := uuid.Parse(toStr)
	if err != nil {
		auth.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	key, _ := moneyjson.String(raw["idempotency_key"])
	note, _ := moneyjson.String(raw["note"])
	res, err := h.svc.Move(r.Context(), customerID, fromID, toID, amount, key, note)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	if !res.Idempotent {
		h.recordMove(r, customerID, res)
	}
	status := http.StatusCreated
	if res.Idempotent {
		status = http.StatusOK
	}
	writeJSON(w, status, map[string]any{
		"from":    toAccountBody(res.From),
		"to":      toAccountBody(res.To),
		"journal": toJournalBody(res.Journal),
		"replay":  res.Idempotent,
	})
}

func (h *Handler) writeStatusChange(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context, customerID, accountID uuid.UUID) (Account, error), action string) {
	customerID, acctID, ok := pathAccount(w, r)
	if !ok {
		return
	}
	before, err := h.svc.GetOwned(r.Context(), customerID, acctID)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	acct, err := fn(r.Context(), customerID, acctID)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	if before.Status != acct.Status {
		h.recordStatus(r, customerID, action, acct)
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
	acct, err := h.svc.GetOwned(r.Context(), customerID, acctID)
	if err != nil {
		writeAccountError(w, err)
		return
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
	if names == nil {
		names = map[string]string{}
	}
	if list, err := h.svc.List(r.Context(), customerID); err == nil {
		for _, a := range list {
			if a.IsJar() {
				names[a.AccountNumber] = a.DisplayName()
			}
		}
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
			"currency":     string(acct.Currency),
		}
		if e.Note != "" {
			item["note"] = e.Note
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
	details := map[string]string{}
	switch a.Currency {
	case currency.GBP:
		details["sort_code"] = currency.SortDisplay()
		details["account"] = currency.LocalAccount(a.AccountNumber)
		details["bic"] = currency.BIC
	case currency.EUR:
		details["iban"] = a.AccountNumber
		details["bic"] = currency.BIC
	case currency.USD:
		details["routing_number"] = currency.RoutingABA
		details["account"] = currency.LocalAccount(a.AccountNumber)
	default:
		details["account"] = currency.LocalAccount(a.AccountNumber)
	}
	return accountBody{
		ID:                     a.ID.String(),
		Currency:               string(a.Currency),
		AccountNumber:          a.AccountNumber,
		AccountNumberFormatted: currency.Format(a.AccountNumber),
		Product:                string(a.Product),
		Label:                  a.Label,
		Status:                 string(a.Status),
		BalanceCents:           a.BalanceCents,
		OpenedAt:               a.OpenedAt.UTC().Format(time.RFC3339),
		Details:                details,
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

func (h *Handler) recordMoney(r *http.Request, customerID uuid.UUID, action string, acct Account, journal ledger.Journal, amountCents int64) {
	if h.audit == nil {
		return
	}
	_ = h.audit.InsertAudit(r.Context(), auth.AuditRecord{
		ID:        uuid.New(),
		ActorID:   &customerID,
		Action:    action,
		IP:        audit.ClientIP(r),
		UserAgent: r.UserAgent(),
		Metadata: map[string]string{
			"account_id":     acct.ID.String(),
			"account_number": acct.AccountNumber,
			"currency":       string(acct.Currency),
			"amount_cents":   strconv.FormatInt(amountCents, 10),
			"journal_id":     journal.ID.String(),
		},
	})
}

func (h *Handler) recordMove(r *http.Request, customerID uuid.UUID, res MoveResult) {
	if h.audit == nil {
		return
	}
	_ = h.audit.InsertAudit(r.Context(), auth.AuditRecord{
		ID:        uuid.New(),
		ActorID:   &customerID,
		Action:    audit.Move,
		IP:        audit.ClientIP(r),
		UserAgent: r.UserAgent(),
		Metadata: map[string]string{
			"from_account_id": res.From.ID.String(),
			"to_account_id":   res.To.ID.String(),
			"account_id":      res.From.ID.String(),
			"currency":        string(res.From.Currency),
			"amount_cents":    strconv.FormatInt(res.AmountCents, 10),
			"journal_id":      res.Journal.ID.String(),
		},
	})
}

func (h *Handler) recordStatus(r *http.Request, customerID uuid.UUID, action string, acct Account) {
	if h.audit == nil {
		return
	}
	_ = h.audit.InsertAudit(r.Context(), auth.AuditRecord{
		ID:        uuid.New(),
		ActorID:   &customerID,
		Action:    action,
		IP:        audit.ClientIP(r),
		UserAgent: r.UserAgent(),
		Metadata: map[string]string{
			"account_id":     acct.ID.String(),
			"account_number": acct.AccountNumber,
			"currency":       string(acct.Currency),
			"status":         string(acct.Status),
		},
	})
}

func writeAccountError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidRequest), errors.Is(err, ErrInvalidAmount), errors.Is(err, ErrIdempotency), errors.Is(err, ledger.ErrInvalidAmount), errors.Is(err, ledger.ErrUnbalanced), errors.Is(err, ledger.ErrIdempotency), errors.Is(err, ledger.ErrInvalidNote):
		auth.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
	case errors.Is(err, ErrExists):
		auth.WriteError(w, http.StatusConflict, "account_exists", "this currency is already open")
	case errors.Is(err, ErrCurrency):
		auth.WriteError(w, http.StatusBadRequest, "currency_mismatch", "send only works in the same currency")
	case errors.Is(err, ErrRates):
		auth.WriteError(w, http.StatusServiceUnavailable, "rates_unavailable", "live rates are unavailable")
	case errors.Is(err, ErrNotFound):
		auth.WriteError(w, http.StatusNotFound, "account_not_found", "account not found")
	case errors.Is(err, ErrFrozen):
		auth.WriteError(w, http.StatusForbidden, "account_frozen", "account cannot move money")
	case errors.Is(err, ErrClosed):
		auth.WriteError(w, http.StatusForbidden, "account_closed", "account is closed")
	case errors.Is(err, ErrHasBalance):
		auth.WriteError(w, http.StatusConflict, "account_has_balance", "withdraw remaining funds before closing")
	case errors.Is(err, ErrHasJars):
		auth.WriteError(w, http.StatusConflict, "jars_open", "move jar balances out and close them first")
	case errors.Is(err, ErrJar):
		auth.WriteError(w, http.StatusBadRequest, "jar_account", "jars only move money inside your own balances")
	case errors.Is(err, ErrJarLimit):
		auth.WriteError(w, http.StatusConflict, "jar_limit", "this currency already has the maximum number of jars")
	case errors.Is(err, ErrNeedSpend):
		auth.WriteError(w, http.StatusConflict, "need_spend", "open this currency first")
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

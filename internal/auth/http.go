package auth

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

const refreshCookie = "refresh_token"

type Handler struct {
	svc          *Service
	cookieSecure bool
	loginLimit   *Limiter
}

func NewHandler(svc *Service, cookieSecure bool) *Handler {
	return &Handler{
		svc:          svc,
		cookieSecure: cookieSecure,
		loginLimit:   NewLimiter(),
	}
}

type credentialsBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type customerBody struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type sessionBody struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresIn   int          `json:"expires_in"`
	Customer    customerBody `json:"customer"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var body credentialsBody
	if err := decodeJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	sess, err := h.svc.Register(r.Context(), RegisterInput{
		Email:     body.Email,
		Password:  body.Password,
		IP:        clientIP(r),
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		writeAuthError(w, err)
		return
	}
	h.setRefreshCookie(w, sess)
	writeJSON(w, http.StatusCreated, toSessionBody(sess))
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var body credentialsBody
	if err := decodeJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	ip := clientIP(r)
	email := strings.ToLower(strings.TrimSpace(body.Email))
	if !h.loginLimit.Allow(ip+"\x00"+email, 10, 15*time.Minute) {
		WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many attempts, try again later")
		return
	}
	sess, err := h.svc.Login(r.Context(), LoginInput{
		Email:     body.Email,
		Password:  body.Password,
		IP:        ip,
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		writeAuthError(w, err)
		return
	}
	h.setRefreshCookie(w, sess)
	writeJSON(w, http.StatusOK, toSessionBody(sess))
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(refreshCookie)
	if err != nil || c.Value == "" {
		WriteError(w, http.StatusUnauthorized, "unauthorized", "please sign in again")
		return
	}
	sess, err := h.svc.Refresh(r.Context(), c.Value, clientIP(r), r.UserAgent())
	if err != nil {
		h.clearRefreshCookie(w)
		writeAuthError(w, err)
		return
	}
	h.setRefreshCookie(w, sess)
	writeJSON(w, http.StatusOK, toSessionBody(sess))
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var actor *uuid.UUID
	if header := r.Header.Get("Authorization"); strings.HasPrefix(header, "Bearer ") {
		if id, err := h.svc.ParseAccess(strings.TrimPrefix(header, "Bearer ")); err == nil {
			actor = &id
		}
	}
	plain := ""
	if c, err := r.Cookie(refreshCookie); err == nil {
		plain = c.Value
	}
	_ = h.svc.Logout(r.Context(), plain, clientIP(r), r.UserAgent(), actor)
	h.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	id, ok := CustomerIDFrom(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "unauthorized", "please sign in again")
		return
	}
	cust, err := h.svc.Me(r.Context(), id)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"customer": customerBody{
			ID:        cust.ID.String(),
			Email:     cust.Email,
			Status:    string(cust.Status),
			CreatedAt: cust.CreatedAt.UTC().Format(time.RFC3339),
		},
	})
}

func (h *Handler) Bearer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			WriteError(w, http.StatusUnauthorized, "unauthorized", "please sign in again")
			return
		}
		id, err := h.svc.ParseAccess(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			WriteError(w, http.StatusUnauthorized, "unauthorized", "please sign in again")
			return
		}
		next.ServeHTTP(w, r.WithContext(WithCustomerID(r.Context(), id)))
	})
}

func (h *Handler) setRefreshCookie(w http.ResponseWriter, sess Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookie,
		Value:    sess.RefreshToken,
		Path:     "/api/v1/auth",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.cookieSecure,
		Expires:  sess.RefreshExp,
	})
}

func (h *Handler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookie,
		Value:    "",
		Path:     "/api/v1/auth",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.cookieSecure,
		MaxAge:   -1,
	})
}

func toSessionBody(sess Session) sessionBody {
	return sessionBody{
		AccessToken: sess.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   sess.ExpiresIn,
		Customer: customerBody{
			ID:        sess.Customer.ID.String(),
			Email:     sess.Customer.Email,
			Status:    string(sess.Customer.Status),
			CreatedAt: sess.Customer.CreatedAt.UTC().Format(time.RFC3339),
		},
	}
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidRequest):
		WriteError(w, http.StatusBadRequest, "invalid_request", "email or password does not meet requirements")
	case errors.Is(err, ErrEmailTaken):
		WriteError(w, http.StatusConflict, "email_taken", "that email is already registered")
	case errors.Is(err, ErrInvalidCredentials):
		WriteError(w, http.StatusUnauthorized, "invalid_credentials", "incorrect email or password")
	case errors.Is(err, ErrLocked):
		WriteError(w, http.StatusForbidden, "account_locked", "account is locked")
	case errors.Is(err, ErrUnauthorized), errors.Is(err, ErrRefreshInvalid):
		WriteError(w, http.StatusUnauthorized, "unauthorized", "please sign in again")
	default:
		WriteError(w, http.StatusInternalServerError, "internal", "service unavailable")
	}
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

package httpapi

import (
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"bank/internal/account"
	"bank/internal/auth"
	"bank/internal/payee"
	"bank/internal/transfer"
)

func New(authH *auth.Handler, accountH *account.Handler, transferH *transfer.Handler, payeeH *payee.Handler, webDist string, log *slog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(maxBody(1 << 20))
	r.Use(accessLog(log))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authH.Register)
			r.Post("/login", authH.Login)
			r.Post("/refresh", authH.Refresh)
			r.Post("/logout", authH.Logout)
		})
		r.Group(func(r chi.Router) {
			r.Use(authH.Bearer)
			r.Get("/me", authH.Me)
			r.Get("/audit", authH.ListAudit)
			r.Post("/accounts", accountH.Open)
			r.Get("/accounts", accountH.List)
			r.Get("/accounts/{id}", accountH.Get)
			r.Post("/accounts/{id}/funding", accountH.Fund)
			r.Post("/accounts/{id}/withdrawals", accountH.Withdraw)
			r.Post("/accounts/{id}/freeze", accountH.Freeze)
			r.Post("/accounts/{id}/unfreeze", accountH.Unfreeze)
			r.Post("/accounts/{id}/close", accountH.Close)
			r.Get("/accounts/{id}/activity", accountH.Activity)
			r.Post("/transfers", transferH.Create)
			r.Get("/payees", payeeH.List)
			r.Post("/payees", payeeH.Create)
			r.Delete("/payees/{id}", payeeH.Delete)
		})
	})

	if info, err := os.Stat(webDist); err == nil && info.IsDir() {
		files := spaHandler(webDist)
		r.NotFound(func(w http.ResponseWriter, req *http.Request) {
			if strings.HasPrefix(req.URL.Path, "/api/") {
				auth.WriteError(w, http.StatusNotFound, "not_found", "not found")
				return
			}
			files.ServeHTTP(w, req)
		})
	}

	return r
}

func maxBody(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, n)
			next.ServeHTTP(w, r)
		})
	}
}

func accessLog(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			log.Info("http",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"ms", time.Since(start).Milliseconds(),
			)
		})
	}
}

func spaHandler(dist string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		target := filepath.Join(dist, rel)
		absDist, err := filepath.Abs(dist)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		absTarget, err := filepath.Abs(target)
		if err != nil || (absTarget != absDist && !strings.HasPrefix(absTarget, absDist+string(filepath.Separator))) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if info, err := os.Stat(absTarget); err == nil && !info.IsDir() {
			http.ServeFile(w, r, absTarget)
			return
		}
		http.ServeFile(w, r, filepath.Join(absDist, "index.html"))
	})
}

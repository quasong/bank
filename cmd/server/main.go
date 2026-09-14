package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bank/internal/account"
	"bank/internal/auth"
	"bank/internal/config"
	"bank/internal/db"
	"bank/internal/httpapi"
	"bank/internal/transfer"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if err := run(log); err != nil {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := db.Migrate(cfg.DatabaseURL, cfg.MigrationsDir); err != nil {
		return err
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	store := db.NewStore(pool)
	tokens := auth.NewTokenManager(cfg.JWTSecret, 15*time.Minute)
	authSvc := auth.NewService(store, auth.NewArgon2Hasher(), tokens, 7*24*time.Hour)
	authH := auth.NewHandler(authSvc, cfg.CookieSecure)
	accountSvc := account.NewService(store)
	accountH := account.NewHandler(accountSvc)
	transferH := transfer.NewHandler(transfer.NewService(store))
	handler := httpapi.New(authH, accountH, transferH, cfg.WebDist, log)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("listen", "addr", cfg.HTTPAddr)
		errCh <- srv.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case sig := <-stop:
		log.Info("shutdown", "signal", sig.String())
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	}
}

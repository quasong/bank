package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	HTTPAddr      string
	DatabaseURL   string
	JWTSecret     string
	CookieSecure  bool
	MigrationsDir string
	WebDist       string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:      env("HTTP_ADDR", ":8080"),
		DatabaseURL:   env("DATABASE_URL", "postgres://bank:bank@localhost:5432/bank?sslmode=disable"),
		JWTSecret:     env("JWT_SECRET", "dev-change-me-use-at-least-32-bytes"),
		CookieSecure:  env("COOKIE_SECURE", "false") == "true",
		MigrationsDir: env("MIGRATIONS_DIR", "migrations"),
		WebDist:       env("WEB_DIST", "web/dist"),
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least 32 bytes")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

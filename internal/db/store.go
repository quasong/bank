package db

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"bank/internal/auth"
	"bank/internal/customer"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) CreateCustomer(ctx context.Context, id uuid.UUID, email, passwordHash string, status customer.Status) (customer.Customer, error) {
	const q = `
		INSERT INTO customers (id, email, password_hash, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, email, status, created_at`
	var c customer.Customer
	err := s.pool.QueryRow(ctx, q, id, email, passwordHash, string(status)).Scan(
		&c.ID, &c.Email, &c.Status, &c.CreatedAt,
	)
	if err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			return customer.Customer{}, auth.ErrEmailTaken
		}
		return customer.Customer{}, err
	}
	return c, nil
}

func (s *Store) GetCustomerByEmail(ctx context.Context, email string) (auth.CustomerRecord, error) {
	return s.getCustomer(ctx, s.pool, `
		SELECT id, email, password_hash, status, created_at
		FROM customers WHERE email = $1`, email)
}

func (s *Store) GetCustomerByID(ctx context.Context, id uuid.UUID) (auth.CustomerRecord, error) {
	return s.getCustomer(ctx, s.pool, `
		SELECT id, email, password_hash, status, created_at
		FROM customers WHERE id = $1`, id)
}

func (s *Store) InsertRefreshToken(ctx context.Context, rec auth.NewRefresh) error {
	const q = `
		INSERT INTO refresh_tokens (id, customer_id, token_hash, expires_at, ip, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := s.pool.Exec(ctx, q, rec.ID, rec.CustomerID, rec.TokenHash, rec.ExpiresAt, rec.IP, rec.UserAgent)
	return err
}

func (s *Store) RotateRefreshToken(ctx context.Context, presentedHash string, next auth.NewRefresh, now time.Time) (auth.RefreshRecord, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return auth.RefreshRecord{}, err
	}
	defer tx.Rollback(ctx)

	const lockQ = `
		SELECT id, customer_id, token_hash, expires_at, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1
		FOR UPDATE`
	var old auth.RefreshRecord
	err = tx.QueryRow(ctx, lockQ, presentedHash).Scan(
		&old.ID, &old.CustomerID, &old.TokenHash, &old.ExpiresAt, &old.RevokedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.RefreshRecord{}, auth.ErrNotFound
	}
	if err != nil {
		return auth.RefreshRecord{}, err
	}
	if old.RevokedAt != nil || !old.ExpiresAt.After(now) {
		return old, auth.ErrRefreshInvalid
	}

	next.CustomerID = old.CustomerID
	const insertQ = `
		INSERT INTO refresh_tokens (id, customer_id, token_hash, expires_at, ip, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6)`
	if _, err := tx.Exec(ctx, insertQ, next.ID, next.CustomerID, next.TokenHash, next.ExpiresAt, next.IP, next.UserAgent); err != nil {
		return auth.RefreshRecord{}, err
	}
	const revokeQ = `
		UPDATE refresh_tokens
		SET revoked_at = $2, replaced_by = $3
		WHERE id = $1 AND revoked_at IS NULL`
	if _, err := tx.Exec(ctx, revokeQ, old.ID, now, next.ID); err != nil {
		return auth.RefreshRecord{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return auth.RefreshRecord{}, err
	}
	return old, nil
}

func (s *Store) RevokeRefreshTokenByHash(ctx context.Context, hash string, now time.Time) error {
	const q = `
		UPDATE refresh_tokens
		SET revoked_at = $2
		WHERE token_hash = $1 AND revoked_at IS NULL`
	_, err := s.pool.Exec(ctx, q, hash, now)
	return err
}

func (s *Store) InsertAudit(ctx context.Context, rec auth.AuditRecord) error {
	meta := []byte("{}")
	if rec.Metadata != nil {
		if b, err := json.Marshal(rec.Metadata); err == nil {
			meta = b
		}
	}
	const q = `
		INSERT INTO audit_logs (id, actor_id, action, ip, user_agent, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := s.pool.Exec(ctx, q, rec.ID, rec.ActorID, rec.Action, rec.IP, rec.UserAgent, meta)
	return err
}

func (s *Store) ListAudit(ctx context.Context, actorID uuid.UUID, limit, offset int32) ([]auth.AuditRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, action, ip, user_agent, metadata, created_at
		FROM audit_logs
		WHERE actor_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3`, actorID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]auth.AuditRecord, 0)
	for rows.Next() {
		var rec auth.AuditRecord
		var meta []byte
		if err := rows.Scan(&rec.ID, &rec.Action, &rec.IP, &rec.UserAgent, &meta, &rec.CreatedAt); err != nil {
			return nil, err
		}
		rec.ActorID = &actorID
		if len(meta) > 0 {
			_ = json.Unmarshal(meta, &rec.Metadata)
		}
		if rec.Metadata == nil {
			rec.Metadata = map[string]string{}
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (s *Store) getCustomer(ctx context.Context, q querier, sql string, arg any) (auth.CustomerRecord, error) {
	var rec auth.CustomerRecord
	err := q.QueryRow(ctx, sql, arg).Scan(
		&rec.Customer.ID,
		&rec.Customer.Email,
		&rec.PasswordHash,
		&rec.Customer.Status,
		&rec.Customer.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.CustomerRecord{}, auth.ErrNotFound
	}
	if err != nil {
		return auth.CustomerRecord{}, err
	}
	return rec, nil
}

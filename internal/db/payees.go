package db

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"bank/internal/payee"
)

func (s *Store) ListPayees(ctx context.Context, customerID uuid.UUID) ([]payee.Payee, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, customer_id, account_number, display_name, created_at, last_used_at
		FROM payees
		WHERE customer_id = $1
		ORDER BY last_used_at DESC, created_at DESC`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]payee.Payee, 0)
	for rows.Next() {
		var p payee.Payee
		if err := rows.Scan(&p.ID, &p.CustomerID, &p.AccountNumber, &p.DisplayName, &p.CreatedAt, &p.LastUsedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) UpsertPayee(ctx context.Context, customerID uuid.UUID, accountNumber, displayName string, overwriteName bool) (payee.Payee, error) {
	var p payee.Payee
	err := s.pool.QueryRow(ctx, `
		INSERT INTO payees (id, customer_id, account_number, display_name)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (customer_id, account_number) DO UPDATE
		SET last_used_at = now(),
		    display_name = CASE WHEN $5 THEN EXCLUDED.display_name ELSE payees.display_name END
		RETURNING id, customer_id, account_number, display_name, created_at, last_used_at`,
		uuid.New(), customerID, accountNumber, displayName, overwriteName,
	).Scan(&p.ID, &p.CustomerID, &p.AccountNumber, &p.DisplayName, &p.CreatedAt, &p.LastUsedAt)
	if err != nil {
		return payee.Payee{}, err
	}
	return p, nil
}

func (s *Store) RenamePayee(ctx context.Context, customerID, id uuid.UUID, displayName string, provided bool) (payee.Payee, error) {
	var p payee.Payee
	err := s.pool.QueryRow(ctx, `
		UPDATE payees
		SET display_name = CASE WHEN $3 THEN $4 ELSE account_number END
		WHERE id = $1 AND customer_id = $2
		RETURNING id, customer_id, account_number, display_name, created_at, last_used_at`,
		id, customerID, provided, displayName,
	).Scan(&p.ID, &p.CustomerID, &p.AccountNumber, &p.DisplayName, &p.CreatedAt, &p.LastUsedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return payee.Payee{}, payee.ErrNotFound
	}
	if err != nil {
		return payee.Payee{}, err
	}
	return p, nil
}

func (s *Store) DeletePayee(ctx context.Context, customerID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM payees WHERE id = $1 AND customer_id = $2`, id, customerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return payee.ErrNotFound
	}
	return nil
}

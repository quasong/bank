-- name: ListPayees :many
SELECT id, customer_id, account_number, display_name, created_at, last_used_at
FROM payees
WHERE customer_id = $1
ORDER BY last_used_at DESC, created_at DESC;

-- name: UpsertPayee :one
INSERT INTO payees (id, customer_id, account_number, display_name)
VALUES ($1, $2, $3, $4)
ON CONFLICT (customer_id, account_number) DO UPDATE
SET last_used_at = now(),
    display_name = CASE WHEN $5::boolean THEN EXCLUDED.display_name ELSE payees.display_name END
RETURNING id, customer_id, account_number, display_name, created_at, last_used_at;

-- name: RenamePayee :one
UPDATE payees
SET display_name = CASE WHEN $3::boolean THEN $4 ELSE account_number END
WHERE id = $1 AND customer_id = $2
RETURNING id, customer_id, account_number, display_name, created_at, last_used_at;

-- name: DeletePayee :execrows
DELETE FROM payees
WHERE id = $1 AND customer_id = $2;

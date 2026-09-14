-- name: InsertAccount :one
INSERT INTO accounts (id, customer_id, account_number, status, balance_cents)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, customer_id, account_number, status, balance_cents, opened_at;

-- name: GetAccountByID :one
SELECT id, customer_id, account_number, status, balance_cents, opened_at
FROM accounts
WHERE id = $1;

-- name: GetAccountByCustomer :one
SELECT id, customer_id, account_number, status, balance_cents, opened_at
FROM accounts
WHERE customer_id = $1;

-- name: GetAccountByNumber :one
SELECT id, customer_id, account_number, status, balance_cents, opened_at
FROM accounts
WHERE account_number = $1;

-- name: ListAccountsByCustomer :many
SELECT id, customer_id, account_number, status, balance_cents, opened_at
FROM accounts
WHERE customer_id = $1
ORDER BY opened_at;

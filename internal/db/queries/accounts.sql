-- name: InsertAccount :one
INSERT INTO accounts (id, customer_id, currency, account_number, status, balance_cents)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, customer_id, currency, account_number, status, balance_cents, opened_at;

-- name: GetAccountByID :one
SELECT id, customer_id, currency, account_number, status, balance_cents, opened_at
FROM accounts
WHERE id = $1;

-- name: GetAccountByCustomerCurrency :one
SELECT id, customer_id, currency, account_number, status, balance_cents, opened_at
FROM accounts
WHERE customer_id = $1 AND currency = $2;

-- name: GetAccountByNumber :one
SELECT id, customer_id, currency, account_number, status, balance_cents, opened_at
FROM accounts
WHERE account_number = $1;

-- name: ListAccountsByCustomer :many
SELECT id, customer_id, currency, account_number, status, balance_cents, opened_at
FROM accounts
WHERE customer_id = $1
ORDER BY opened_at;

-- name: LockAccountByID :one
SELECT a.id, a.customer_id, a.currency, a.account_number, a.status, a.balance_cents, a.opened_at, la.id AS ledger_id
FROM accounts a
JOIN ledger_accounts la ON la.account_id = a.id
WHERE a.id = $1
FOR UPDATE OF a;

-- name: UpdateAccountStatus :exec
UPDATE accounts SET status = $2 WHERE id = $1;

-- name: InsertCustomer :one
INSERT INTO customers (id, email, password_hash, status)
VALUES ($1, $2, $3, $4)
RETURNING id, email, password_hash, status, created_at, updated_at;

-- name: GetCustomerByEmail :one
SELECT id, email, password_hash, status, created_at, updated_at
FROM customers
WHERE email = $1;

-- name: GetCustomerByID :one
SELECT id, email, password_hash, status, created_at, updated_at
FROM customers
WHERE id = $1;

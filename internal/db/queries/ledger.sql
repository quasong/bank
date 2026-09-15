-- name: InsertLedgerAccount :exec
INSERT INTO ledger_accounts (id, name, kind, account_id)
VALUES ($1, $2, $3, $4);

-- name: GetLedgerAccountByDeposit :one
SELECT id, name, kind, account_id, created_at
FROM ledger_accounts
WHERE account_id = $1;

-- name: InsertJournal :one
INSERT INTO journals (id, description, note, kind, idempotency_key)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, created_at, description, note, kind, idempotency_key;

-- name: GetJournalByIdempotencyKey :one
SELECT id, created_at, description, note, kind, idempotency_key
FROM journals
WHERE idempotency_key = $1;

-- name: InsertJournalLine :exec
INSERT INTO journal_lines (id, journal_id, ledger_account_id, side, amount_cents)
VALUES ($1, $2, $3, $4, $5);

-- name: ListJournalLines :many
SELECT id, journal_id, ledger_account_id, side, amount_cents
FROM journal_lines
WHERE journal_id = $1
ORDER BY id;

-- name: ApplyAccountDelta :execrows
UPDATE accounts
SET balance_cents = balance_cents + $2
WHERE id = $1
  AND status = 'active'
  AND balance_cents + $2 >= 0;

-- name: ListActivity :many
SELECT
    j.id AS journal_id,
    j.created_at,
    j.kind,
    j.description,
    jl.side,
    jl.amount_cents,
    j.note,
    (
        SELECT a.account_number
        FROM journal_lines ojl
        JOIN ledger_accounts ola ON ola.id = ojl.ledger_account_id
        JOIN accounts a ON a.id = ola.account_id
        WHERE ojl.journal_id = jl.journal_id
          AND ojl.id <> jl.id
        LIMIT 1
    ) AS counterparty_account_number
FROM journal_lines jl
JOIN journals j ON j.id = jl.journal_id
JOIN ledger_accounts la ON la.id = jl.ledger_account_id
WHERE la.account_id = $1
ORDER BY j.created_at DESC, jl.id DESC
LIMIT $2 OFFSET $3;

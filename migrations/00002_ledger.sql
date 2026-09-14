-- +goose Up
CREATE TABLE accounts (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL UNIQUE REFERENCES customers (id),
    account_number TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('active', 'frozen', 'closed')),
    balance_cents BIGINT NOT NULL DEFAULT 0 CHECK (balance_cents >= 0),
    opened_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ledger_accounts (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('asset', 'liability')),
    account_id UUID UNIQUE REFERENCES accounts (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE journals (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    description TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('funding', 'transfer')),
    idempotency_key TEXT NOT NULL UNIQUE
);

CREATE TABLE journal_lines (
    id UUID PRIMARY KEY,
    journal_id UUID NOT NULL REFERENCES journals (id),
    ledger_account_id UUID NOT NULL REFERENCES ledger_accounts (id),
    side TEXT NOT NULL CHECK (side IN ('debit', 'credit')),
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0)
);

CREATE INDEX journal_lines_journal_id_idx ON journal_lines (journal_id);
CREATE INDEX journal_lines_ledger_account_id_idx ON journal_lines (ledger_account_id);
CREATE INDEX journals_created_at_idx ON journals (created_at DESC);

INSERT INTO ledger_accounts (id, name, kind, account_id)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    'Vault cash',
    'asset',
    NULL
);

-- +goose Down
DROP TABLE IF EXISTS journal_lines;
DROP TABLE IF EXISTS journals;
DROP TABLE IF EXISTS ledger_accounts;
DROP TABLE IF EXISTS accounts;

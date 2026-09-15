CREATE TABLE customers (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'locked')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers (id),
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    replaced_by UUID REFERENCES refresh_tokens (id),
    ip TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX refresh_tokens_customer_id_idx ON refresh_tokens (customer_id);

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY,
    actor_id UUID REFERENCES customers (id),
    action TEXT NOT NULL,
    ip TEXT,
    user_agent TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX audit_logs_actor_id_created_at_idx ON audit_logs (actor_id, created_at DESC);

CREATE TABLE accounts (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers (id),
    currency TEXT NOT NULL CHECK (currency IN ('USD', 'EUR', 'GBP')),
    account_number TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('active', 'frozen', 'closed')),
    balance_cents BIGINT NOT NULL DEFAULT 0 CHECK (balance_cents >= 0),
    opened_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (customer_id, currency),
    CHECK (
        (currency = 'USD' AND account_number ~ '^[0-9]{8}$')
        OR (currency = 'GBP' AND account_number ~ '^040004[0-9]{8}$')
        OR (currency = 'EUR' AND account_number ~ '^GB[0-9]{2}THEB040004[0-9]{8}$')
    )
);

CREATE TABLE ledger_accounts (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('asset', 'liability')),
    currency TEXT NOT NULL CHECK (currency IN ('USD', 'EUR', 'GBP')),
    account_id UUID UNIQUE REFERENCES accounts (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE journals (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    description TEXT NOT NULL,
    note TEXT NOT NULL DEFAULT '' CHECK (char_length(note) <= 40),
    kind TEXT NOT NULL CHECK (kind IN ('funding', 'transfer', 'withdrawal', 'fx')),
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

INSERT INTO ledger_accounts (id, name, kind, currency, account_id)
VALUES
    ('11111111-1111-1111-1111-111111111111', 'Vault cash USD', 'asset', 'USD', NULL),
    ('22222222-2222-2222-2222-222222222222', 'Vault cash EUR', 'asset', 'EUR', NULL),
    ('33333333-3333-3333-3333-333333333333', 'Vault cash GBP', 'asset', 'GBP', NULL);

CREATE TABLE payees (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers (id),
    account_number TEXT NOT NULL CHECK (
        account_number ~ '^[0-9]{8}$'
        OR account_number ~ '^040004[0-9]{8}$'
        OR account_number ~ '^GB[0-9]{2}THEB040004[0-9]{8}$'
    ),
    display_name TEXT NOT NULL CHECK (char_length(display_name) BETWEEN 1 AND 40),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (customer_id, account_number)
);

CREATE INDEX payees_customer_id_last_used_at_idx ON payees (customer_id, last_used_at DESC);


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
    currency TEXT NOT NULL CHECK (currency IN (
        'USD', 'EUR', 'GBP', 'AUD', 'BGN', 'BRL', 'CAD', 'CHF', 'CNY', 'CZK',
        'DKK', 'HKD', 'ILS', 'INR', 'MXN', 'MYR', 'NOK', 'NZD', 'PHP', 'PLN',
        'RON', 'SEK', 'SGD', 'THB', 'TRY', 'ZAR'
    )),
    account_number TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('active', 'frozen', 'closed')),
    balance_cents BIGINT NOT NULL DEFAULT 0 CHECK (balance_cents >= 0),
    opened_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (customer_id, currency),
    CHECK (
        (currency = 'USD' AND account_number ~ '^121000248[0-9]{8}$')
        OR (currency = 'GBP' AND account_number ~ '^040004[0-9]{8}$')
        OR (currency = 'EUR' AND account_number ~ '^GB[0-9]{2}THEB040004[0-9]{8}$')
        OR (
            currency NOT IN ('USD', 'EUR', 'GBP')
            AND account_number ~ '^[A-Z]{3}[0-9]{8}$'
            AND left(account_number, 3) = currency
        )
    )
);

CREATE TABLE ledger_accounts (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('asset', 'liability')),
    currency TEXT NOT NULL CHECK (currency IN (
        'USD', 'EUR', 'GBP', 'AUD', 'BGN', 'BRL', 'CAD', 'CHF', 'CNY', 'CZK',
        'DKK', 'HKD', 'ILS', 'INR', 'MXN', 'MYR', 'NOK', 'NZD', 'PHP', 'PLN',
        'RON', 'SEK', 'SGD', 'THB', 'TRY', 'ZAR'
    )),
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
    ('33333333-3333-3333-3333-333333333333', 'Vault cash GBP', 'asset', 'GBP', NULL),
    ('237b5673-edd9-5c45-8ba1-75983e0bc1fa', 'Vault cash AUD', 'asset', 'AUD', NULL),
    ('18881fce-981d-5a9b-8979-447abfd5d88a', 'Vault cash BGN', 'asset', 'BGN', NULL),
    ('73f04e5f-ad2f-5699-864b-b7c272d00b83', 'Vault cash BRL', 'asset', 'BRL', NULL),
    ('0436293e-fc79-564b-a90c-5ec42dc603df', 'Vault cash CAD', 'asset', 'CAD', NULL),
    ('dbc51fcc-790a-54c7-957f-2d2cd94eb066', 'Vault cash CHF', 'asset', 'CHF', NULL),
    ('2dcf96bd-9d8d-5977-8140-799c639f0b30', 'Vault cash CNY', 'asset', 'CNY', NULL),
    ('18492ab0-0cb2-5cf7-8257-5a58c3bbd332', 'Vault cash CZK', 'asset', 'CZK', NULL),
    ('5dd3b670-69c8-5426-ac45-26d26a07c706', 'Vault cash DKK', 'asset', 'DKK', NULL),
    ('3ca8f31b-5e8c-586a-a979-b4c96e689708', 'Vault cash HKD', 'asset', 'HKD', NULL),
    ('1d9dd38b-8b0b-5910-9a58-7f8ae3418e85', 'Vault cash ILS', 'asset', 'ILS', NULL),
    ('31bcaf06-8439-530c-abb0-41ab07d0be6f', 'Vault cash INR', 'asset', 'INR', NULL),
    ('ac5667a0-e84c-5cf7-8e2b-bcac6f847511', 'Vault cash MXN', 'asset', 'MXN', NULL),
    ('c62ab81c-068f-50b7-a107-1199954b7737', 'Vault cash MYR', 'asset', 'MYR', NULL),
    ('12723183-f93a-5192-bae9-2cfe9005a475', 'Vault cash NOK', 'asset', 'NOK', NULL),
    ('44c43133-e7e3-5eef-b473-325564ba6ce0', 'Vault cash NZD', 'asset', 'NZD', NULL),
    ('ac46d1b4-b2a8-5f74-8c39-1a54292fed81', 'Vault cash PHP', 'asset', 'PHP', NULL),
    ('baecda3f-97b5-56c7-b9da-8a4e379cb945', 'Vault cash PLN', 'asset', 'PLN', NULL),
    ('84ff546e-d01f-57ea-8c92-bbba362a3884', 'Vault cash RON', 'asset', 'RON', NULL),
    ('db2c72c6-cd95-5a38-95b3-0601569a2a0b', 'Vault cash SEK', 'asset', 'SEK', NULL),
    ('e0239868-c5c9-5d09-91b6-d19a3d125766', 'Vault cash SGD', 'asset', 'SGD', NULL),
    ('fccf13ae-a9e6-5499-8ab3-7729e5b93af3', 'Vault cash THB', 'asset', 'THB', NULL),
    ('8c5df232-93c2-5ac1-8978-c1bbff44d4a1', 'Vault cash TRY', 'asset', 'TRY', NULL),
    ('bfa2d0b9-2e0f-533c-b2be-7f8802368467', 'Vault cash ZAR', 'asset', 'ZAR', NULL);

CREATE TABLE payees (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers (id),
    account_number TEXT NOT NULL CHECK (
        account_number ~ '^121000248[0-9]{8}$'
        OR account_number ~ '^040004[0-9]{8}$'
        OR account_number ~ '^GB[0-9]{2}THEB040004[0-9]{8}$'
        OR account_number ~ '^(AUD|BGN|BRL|CAD|CHF|CNY|CZK|DKK|HKD|ILS|INR|MXN|MYR|NOK|NZD|PHP|PLN|RON|SEK|SGD|THB|TRY|ZAR)[0-9]{8}$'
    ),
    display_name TEXT NOT NULL CHECK (char_length(display_name) BETWEEN 1 AND 40),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (customer_id, account_number)
);

CREATE INDEX payees_customer_id_last_used_at_idx ON payees (customer_id, last_used_at DESC);


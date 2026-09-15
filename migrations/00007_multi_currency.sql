-- +goose Up
ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS currency TEXT NOT NULL DEFAULT 'USD';

ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_customer_id_key;

ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_customer_id_currency_key;
ALTER TABLE accounts
    ADD CONSTRAINT accounts_customer_id_currency_key UNIQUE (customer_id, currency);

ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_currency_chk;
ALTER TABLE accounts
    ADD CONSTRAINT accounts_currency_chk CHECK (currency IN ('USD', 'EUR', 'GBP'));

ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_account_number_check;
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_account_number_chk;
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_account_number_digits_chk;
ALTER TABLE accounts
    ADD CONSTRAINT accounts_account_number_chk CHECK (
        (currency = 'USD' AND account_number ~ '^[0-9]{8}$')
        OR (currency = 'GBP' AND account_number ~ '^040004[0-9]{8}$')
        OR (currency = 'EUR' AND account_number ~ '^GB[0-9]{2}THEB040004[0-9]{8}$')
    );

ALTER TABLE ledger_accounts
    ADD COLUMN IF NOT EXISTS currency TEXT NOT NULL DEFAULT 'USD';

ALTER TABLE ledger_accounts DROP CONSTRAINT IF EXISTS ledger_accounts_currency_chk;
ALTER TABLE ledger_accounts
    ADD CONSTRAINT ledger_accounts_currency_chk CHECK (currency IN ('USD', 'EUR', 'GBP'));

UPDATE ledger_accounts
SET name = 'Vault cash USD', currency = 'USD'
WHERE id = '11111111-1111-1111-1111-111111111111';

INSERT INTO ledger_accounts (id, name, kind, account_id, currency)
VALUES
    ('22222222-2222-2222-2222-222222222222', 'Vault cash EUR', 'asset', NULL, 'EUR'),
    ('33333333-3333-3333-3333-333333333333', 'Vault cash GBP', 'asset', NULL, 'GBP')
ON CONFLICT (id) DO NOTHING;

ALTER TABLE journals DROP CONSTRAINT IF EXISTS journals_kind_check;
ALTER TABLE journals
    ADD CONSTRAINT journals_kind_check
    CHECK (kind IN ('funding', 'transfer', 'withdrawal', 'fx'));

ALTER TABLE payees DROP CONSTRAINT IF EXISTS payees_account_number_check;
ALTER TABLE payees
    ADD CONSTRAINT payees_account_number_chk CHECK (
        account_number ~ '^[0-9]{8}$'
        OR account_number ~ '^040004[0-9]{8}$'
        OR account_number ~ '^GB[0-9]{2}THEB040004[0-9]{8}$'
    );

-- +goose Down
ALTER TABLE payees DROP CONSTRAINT IF EXISTS payees_account_number_chk;
ALTER TABLE payees
    ADD CONSTRAINT payees_account_number_check CHECK (account_number ~ '^[0-9]{8}$');

ALTER TABLE journals DROP CONSTRAINT IF EXISTS journals_kind_check;
ALTER TABLE journals
    ADD CONSTRAINT journals_kind_check
    CHECK (kind IN ('funding', 'transfer', 'withdrawal'));

DELETE FROM ledger_accounts
WHERE id IN (
    '22222222-2222-2222-2222-222222222222',
    '33333333-3333-3333-3333-333333333333'
);

ALTER TABLE ledger_accounts DROP CONSTRAINT IF EXISTS ledger_accounts_currency_chk;
ALTER TABLE ledger_accounts DROP COLUMN IF EXISTS currency;

ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_account_number_chk;
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_currency_chk;
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_customer_id_currency_key;
ALTER TABLE accounts DROP COLUMN IF EXISTS currency;
ALTER TABLE accounts
    ADD CONSTRAINT accounts_customer_id_key UNIQUE (customer_id);
ALTER TABLE accounts
    ADD CONSTRAINT accounts_account_number_digits_chk CHECK (account_number ~ '^[0-9]{8}$');

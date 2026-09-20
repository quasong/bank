-- +goose Up
ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS product TEXT NOT NULL DEFAULT 'spend',
    ADD COLUMN IF NOT EXISTS label TEXT NOT NULL DEFAULT '';

ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_product_chk;
ALTER TABLE accounts
    ADD CONSTRAINT accounts_product_chk CHECK (product IN ('spend', 'jar'));

ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_label_chk;
ALTER TABLE accounts
    ADD CONSTRAINT accounts_label_chk CHECK (char_length(label) <= 20);

ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_customer_id_currency_key;
DROP INDEX IF EXISTS accounts_one_spend_idx;
CREATE UNIQUE INDEX accounts_one_spend_idx ON accounts (customer_id, currency) WHERE product = 'spend';

ALTER TABLE journals DROP CONSTRAINT IF EXISTS journals_kind_check;
ALTER TABLE journals
    ADD CONSTRAINT journals_kind_check
    CHECK (kind IN ('funding', 'transfer', 'withdrawal', 'fx', 'move'));

-- +goose Down
ALTER TABLE journals DROP CONSTRAINT IF EXISTS journals_kind_check;
ALTER TABLE journals
    ADD CONSTRAINT journals_kind_check
    CHECK (kind IN ('funding', 'transfer', 'withdrawal', 'fx'));

DROP INDEX IF EXISTS accounts_one_spend_idx;
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_customer_id_currency_key;
ALTER TABLE accounts
    ADD CONSTRAINT accounts_customer_id_currency_key UNIQUE (customer_id, currency);

ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_product_chk;
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_label_chk;
ALTER TABLE accounts DROP COLUMN IF EXISTS product;
ALTER TABLE accounts DROP COLUMN IF EXISTS label;

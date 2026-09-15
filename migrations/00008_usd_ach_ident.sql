-- +goose Up
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_account_number_chk;
ALTER TABLE payees DROP CONSTRAINT IF EXISTS payees_account_number_chk;

UPDATE payees
SET
    display_name = CASE
        WHEN display_name = account_number THEN '121000248' || account_number
        ELSE display_name
    END,
    account_number = '121000248' || account_number
WHERE account_number ~ '^[0-9]{8}$';

UPDATE accounts
SET account_number = '121000248' || account_number
WHERE currency = 'USD' AND account_number ~ '^[0-9]{8}$';

ALTER TABLE accounts
    ADD CONSTRAINT accounts_account_number_chk CHECK (
        (currency = 'USD' AND account_number ~ '^121000248[0-9]{8}$')
        OR (currency = 'GBP' AND account_number ~ '^040004[0-9]{8}$')
        OR (currency = 'EUR' AND account_number ~ '^GB[0-9]{2}THEB040004[0-9]{8}$')
    );

ALTER TABLE payees
    ADD CONSTRAINT payees_account_number_chk CHECK (
        account_number ~ '^121000248[0-9]{8}$'
        OR account_number ~ '^040004[0-9]{8}$'
        OR account_number ~ '^GB[0-9]{2}THEB040004[0-9]{8}$'
    );

-- +goose Down
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_account_number_chk;
ALTER TABLE payees DROP CONSTRAINT IF EXISTS payees_account_number_chk;

UPDATE payees
SET
    display_name = CASE
        WHEN display_name = account_number THEN right(account_number, 8)
        ELSE display_name
    END,
    account_number = right(account_number, 8)
WHERE account_number ~ '^121000248[0-9]{8}$';

UPDATE accounts
SET account_number = right(account_number, 8)
WHERE currency = 'USD' AND account_number ~ '^121000248[0-9]{8}$';

ALTER TABLE accounts
    ADD CONSTRAINT accounts_account_number_chk CHECK (
        (currency = 'USD' AND account_number ~ '^[0-9]{8}$')
        OR (currency = 'GBP' AND account_number ~ '^040004[0-9]{8}$')
        OR (currency = 'EUR' AND account_number ~ '^GB[0-9]{2}THEB040004[0-9]{8}$')
    );

ALTER TABLE payees
    ADD CONSTRAINT payees_account_number_chk CHECK (
        account_number ~ '^[0-9]{8}$'
        OR account_number ~ '^040004[0-9]{8}$'
        OR account_number ~ '^GB[0-9]{2}THEB040004[0-9]{8}$'
    );

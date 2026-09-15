-- +goose Up
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_currency_chk;
ALTER TABLE accounts
    ADD CONSTRAINT accounts_currency_chk CHECK (currency IN (
        'USD', 'EUR', 'GBP', 'AUD', 'BGN', 'BRL', 'CAD', 'CHF', 'CNY', 'CZK',
        'DKK', 'HKD', 'ILS', 'INR', 'MXN', 'MYR', 'NOK', 'NZD', 'PHP', 'PLN',
        'RON', 'SEK', 'SGD', 'THB', 'TRY', 'ZAR'
    ));

ALTER TABLE ledger_accounts DROP CONSTRAINT IF EXISTS ledger_accounts_currency_chk;
ALTER TABLE ledger_accounts
    ADD CONSTRAINT ledger_accounts_currency_chk CHECK (currency IN (
        'USD', 'EUR', 'GBP', 'AUD', 'BGN', 'BRL', 'CAD', 'CHF', 'CNY', 'CZK',
        'DKK', 'HKD', 'ILS', 'INR', 'MXN', 'MYR', 'NOK', 'NZD', 'PHP', 'PLN',
        'RON', 'SEK', 'SGD', 'THB', 'TRY', 'ZAR'
    ));

ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_account_number_chk;
ALTER TABLE accounts
    ADD CONSTRAINT accounts_account_number_chk CHECK (
        (currency = 'USD' AND account_number ~ '^121000248[0-9]{8}$')
        OR (currency = 'GBP' AND account_number ~ '^040004[0-9]{8}$')
        OR (currency = 'EUR' AND account_number ~ '^GB[0-9]{2}THEB040004[0-9]{8}$')
        OR (
            currency NOT IN ('USD', 'EUR', 'GBP')
            AND account_number ~ '^[A-Z]{3}[0-9]{8}$'
            AND left(account_number, 3) = currency
        )
    );

ALTER TABLE payees DROP CONSTRAINT IF EXISTS payees_account_number_chk;
ALTER TABLE payees
    ADD CONSTRAINT payees_account_number_chk CHECK (
        account_number ~ '^121000248[0-9]{8}$'
        OR account_number ~ '^040004[0-9]{8}$'
        OR account_number ~ '^GB[0-9]{2}THEB040004[0-9]{8}$'
        OR account_number ~ '^(AUD|BGN|BRL|CAD|CHF|CNY|CZK|DKK|HKD|ILS|INR|MXN|MYR|NOK|NZD|PHP|PLN|RON|SEK|SGD|THB|TRY|ZAR)[0-9]{8}$'
    );

INSERT INTO ledger_accounts (id, name, kind, account_id, currency)
VALUES
    ('237b5673-edd9-5c45-8ba1-75983e0bc1fa', 'Vault cash AUD', 'asset', NULL, 'AUD'),
    ('18881fce-981d-5a9b-8979-447abfd5d88a', 'Vault cash BGN', 'asset', NULL, 'BGN'),
    ('73f04e5f-ad2f-5699-864b-b7c272d00b83', 'Vault cash BRL', 'asset', NULL, 'BRL'),
    ('0436293e-fc79-564b-a90c-5ec42dc603df', 'Vault cash CAD', 'asset', NULL, 'CAD'),
    ('dbc51fcc-790a-54c7-957f-2d2cd94eb066', 'Vault cash CHF', 'asset', NULL, 'CHF'),
    ('2dcf96bd-9d8d-5977-8140-799c639f0b30', 'Vault cash CNY', 'asset', NULL, 'CNY'),
    ('18492ab0-0cb2-5cf7-8257-5a58c3bbd332', 'Vault cash CZK', 'asset', NULL, 'CZK'),
    ('5dd3b670-69c8-5426-ac45-26d26a07c706', 'Vault cash DKK', 'asset', NULL, 'DKK'),
    ('3ca8f31b-5e8c-586a-a979-b4c96e689708', 'Vault cash HKD', 'asset', NULL, 'HKD'),
    ('1d9dd38b-8b0b-5910-9a58-7f8ae3418e85', 'Vault cash ILS', 'asset', NULL, 'ILS'),
    ('31bcaf06-8439-530c-abb0-41ab07d0be6f', 'Vault cash INR', 'asset', NULL, 'INR'),
    ('ac5667a0-e84c-5cf7-8e2b-bcac6f847511', 'Vault cash MXN', 'asset', NULL, 'MXN'),
    ('c62ab81c-068f-50b7-a107-1199954b7737', 'Vault cash MYR', 'asset', NULL, 'MYR'),
    ('12723183-f93a-5192-bae9-2cfe9005a475', 'Vault cash NOK', 'asset', NULL, 'NOK'),
    ('44c43133-e7e3-5eef-b473-325564ba6ce0', 'Vault cash NZD', 'asset', NULL, 'NZD'),
    ('ac46d1b4-b2a8-5f74-8c39-1a54292fed81', 'Vault cash PHP', 'asset', NULL, 'PHP'),
    ('baecda3f-97b5-56c7-b9da-8a4e379cb945', 'Vault cash PLN', 'asset', NULL, 'PLN'),
    ('84ff546e-d01f-57ea-8c92-bbba362a3884', 'Vault cash RON', 'asset', NULL, 'RON'),
    ('db2c72c6-cd95-5a38-95b3-0601569a2a0b', 'Vault cash SEK', 'asset', NULL, 'SEK'),
    ('e0239868-c5c9-5d09-91b6-d19a3d125766', 'Vault cash SGD', 'asset', NULL, 'SGD'),
    ('fccf13ae-a9e6-5499-8ab3-7729e5b93af3', 'Vault cash THB', 'asset', NULL, 'THB'),
    ('8c5df232-93c2-5ac1-8978-c1bbff44d4a1', 'Vault cash TRY', 'asset', NULL, 'TRY'),
    ('bfa2d0b9-2e0f-533c-b2be-7f8802368467', 'Vault cash ZAR', 'asset', NULL, 'ZAR')
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM ledger_accounts
WHERE id IN (
    '237b5673-edd9-5c45-8ba1-75983e0bc1fa',
    '18881fce-981d-5a9b-8979-447abfd5d88a',
    '73f04e5f-ad2f-5699-864b-b7c272d00b83',
    '0436293e-fc79-564b-a90c-5ec42dc603df',
    'dbc51fcc-790a-54c7-957f-2d2cd94eb066',
    '2dcf96bd-9d8d-5977-8140-799c639f0b30',
    '18492ab0-0cb2-5cf7-8257-5a58c3bbd332',
    '5dd3b670-69c8-5426-ac45-26d26a07c706',
    '3ca8f31b-5e8c-586a-a979-b4c96e689708',
    '1d9dd38b-8b0b-5910-9a58-7f8ae3418e85',
    '31bcaf06-8439-530c-abb0-41ab07d0be6f',
    'ac5667a0-e84c-5cf7-8e2b-bcac6f847511',
    'c62ab81c-068f-50b7-a107-1199954b7737',
    '12723183-f93a-5192-bae9-2cfe9005a475',
    '44c43133-e7e3-5eef-b473-325564ba6ce0',
    'ac46d1b4-b2a8-5f74-8c39-1a54292fed81',
    'baecda3f-97b5-56c7-b9da-8a4e379cb945',
    '84ff546e-d01f-57ea-8c92-bbba362a3884',
    'db2c72c6-cd95-5a38-95b3-0601569a2a0b',
    'e0239868-c5c9-5d09-91b6-d19a3d125766',
    'fccf13ae-a9e6-5499-8ab3-7729e5b93af3',
    '8c5df232-93c2-5ac1-8978-c1bbff44d4a1',
    'bfa2d0b9-2e0f-533c-b2be-7f8802368467'
);

ALTER TABLE payees DROP CONSTRAINT IF EXISTS payees_account_number_chk;
ALTER TABLE payees
    ADD CONSTRAINT payees_account_number_chk CHECK (
        account_number ~ '^121000248[0-9]{8}$'
        OR account_number ~ '^040004[0-9]{8}$'
        OR account_number ~ '^GB[0-9]{2}THEB040004[0-9]{8}$'
    );

ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_account_number_chk;
ALTER TABLE accounts
    ADD CONSTRAINT accounts_account_number_chk CHECK (
        (currency = 'USD' AND account_number ~ '^121000248[0-9]{8}$')
        OR (currency = 'GBP' AND account_number ~ '^040004[0-9]{8}$')
        OR (currency = 'EUR' AND account_number ~ '^GB[0-9]{2}THEB040004[0-9]{8}$')
    );

ALTER TABLE ledger_accounts DROP CONSTRAINT IF EXISTS ledger_accounts_currency_chk;
ALTER TABLE ledger_accounts
    ADD CONSTRAINT ledger_accounts_currency_chk CHECK (currency IN ('USD', 'EUR', 'GBP'));

ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_currency_chk;
ALTER TABLE accounts
    ADD CONSTRAINT accounts_currency_chk CHECK (currency IN ('USD', 'EUR', 'GBP'));

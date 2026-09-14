-- +goose Up
UPDATE accounts AS a
SET account_number = b.num
FROM (
    SELECT
        id,
        lpad(
            (
                COALESCE(
                    (
                        SELECT MAX(account_number::int)
                        FROM accounts
                        WHERE account_number ~ '^[0-9]{8}$'
                    ),
                    0
                ) + row_number() OVER (ORDER BY opened_at, id)
            )::text,
            8,
            '0'
        ) AS num
    FROM accounts
    WHERE account_number !~ '^[0-9]{8}$'
) AS b
WHERE a.id = b.id;

ALTER TABLE accounts
    ADD CONSTRAINT accounts_account_number_digits_chk
    CHECK (account_number ~ '^[0-9]{8}$');

-- +goose Down
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_account_number_digits_chk;

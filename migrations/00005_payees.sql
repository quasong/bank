-- +goose Up
CREATE TABLE payees (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers (id),
    account_number TEXT NOT NULL CHECK (account_number ~ '^[0-9]{8}$'),
    display_name TEXT NOT NULL CHECK (char_length(display_name) BETWEEN 1 AND 40),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (customer_id, account_number)
);

CREATE INDEX payees_customer_id_last_used_at_idx ON payees (customer_id, last_used_at DESC);

-- +goose Down
DROP TABLE IF EXISTS payees;

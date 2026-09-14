-- +goose Up
ALTER TABLE journals DROP CONSTRAINT IF EXISTS journals_kind_check;
ALTER TABLE journals
    ADD CONSTRAINT journals_kind_check
    CHECK (kind IN ('funding', 'transfer', 'withdrawal'));

-- +goose Down
ALTER TABLE journals DROP CONSTRAINT IF EXISTS journals_kind_check;
ALTER TABLE journals
    ADD CONSTRAINT journals_kind_check
    CHECK (kind IN ('funding', 'transfer'));

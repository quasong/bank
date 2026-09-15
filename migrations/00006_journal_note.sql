-- +goose Up
ALTER TABLE journals
    ADD COLUMN note TEXT NOT NULL DEFAULT '';

ALTER TABLE journals
    ADD CONSTRAINT journals_note_len_chk CHECK (char_length(note) <= 40);

-- +goose Down
ALTER TABLE journals DROP CONSTRAINT IF EXISTS journals_note_len_chk;
ALTER TABLE journals DROP COLUMN IF EXISTS note;

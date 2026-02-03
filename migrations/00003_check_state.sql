-- +goose Up
ALTER TABLE checks
    ADD COLUMN status TEXT NOT NULL DEFAULT 'unknown' CHECK (status IN ('unknown', 'up', 'down')),
    ADD COLUMN consecutive_failures INT NOT NULL DEFAULT 0,
    ADD COLUMN status_changed_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- +goose Down
ALTER TABLE checks
    DROP COLUMN status,
    DROP COLUMN consecutive_failures,
    DROP COLUMN status_changed_at;
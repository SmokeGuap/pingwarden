-- +goose Up
CREATE TABLE checks (
    id               BIGSERIAL PRIMARY KEY,
    name             TEXT        NOT NULL,
    url              TEXT        NOT NULL,
    interval_seconds INT         NOT NULL DEFAULT 60 CHECK (interval_seconds >= 10),
    enabled          BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE checks;


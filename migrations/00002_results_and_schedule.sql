-- +goose Up
ALTER TABLE checks ADD COLUMN next_run_at TIMESTAMPTZ NOT NULL DEFAULT now();
CREATE INDEX checks_due_idx ON checks (next_run_at) WHERE enabled;

CREATE TABLE check_results (
    id          BIGSERIAL   PRIMARY KEY,
    check_id    BIGINT      NOT NULL REFERENCES checks(id) ON DELETE CASCADE,
    checked_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    success     BOOLEAN     NOT NULL,
    status_code INT,
    latency_ms  INT         NOT NULL,
    error       TEXT        NOT NULL DEFAULT ''
);
CREATE INDEX check_results_check_time_idx ON check_results (check_id, checked_at DESC);

-- +goose Down
DROP TABLE check_results;
ALTER TABLE checks DROP COLUMN next_run_at;
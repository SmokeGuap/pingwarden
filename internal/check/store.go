package check

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const cols = "id, name, url, interval_seconds, enabled, created_at"

type PostgresStore struct{ pool *pgxpool.Pool }

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool: pool} }

func (s *PostgresStore) Create(ctx context.Context, c *Check) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO checks (name, url, interval_seconds, enabled)
		 VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		c.Name, c.URL, c.IntervalSeconds, c.Enabled,
	).Scan(&c.ID, &c.CreatedAt)
}

func (s *PostgresStore) List(ctx context.Context) ([]Check, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+cols+` FROM checks ORDER BY id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Check])
}

func (s *PostgresStore) Get(ctx context.Context, id int64) (Check, error) {
	var c Check
	err := s.pool.QueryRow(ctx, `SELECT `+cols+` FROM checks WHERE id = $1`, id).
		Scan(&c.ID, &c.Name, &c.URL, &c.IntervalSeconds, &c.Enabled, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Check{}, ErrNotFound
	}
	return c, err
}

func (s *PostgresStore) Update(ctx context.Context, c *Check) error {
	err := s.pool.QueryRow(ctx,
		`UPDATE checks SET name=$2, url=$3, interval_seconds=$4, enabled=$5
		 WHERE id=$1 RETURNING created_at`,
		c.ID, c.Name, c.URL, c.IntervalSeconds, c.Enabled,
	).Scan(&c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (s *PostgresStore) Delete(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM checks WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) ClaimDue(ctx context.Context, limit int) ([]int64, error) {
	rows, err := s.pool.Query(ctx, `
		UPDATE checks
		SET next_run_at = now() + make_interval(secs => interval_seconds)
		WHERE id IN (
			SELECT id FROM checks
			WHERE enabled AND next_run_at <= now()
			ORDER BY next_run_at
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id`, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[int64])
}

func (s *PostgresStore) RecordOutcome(ctx context.Context, id int64, success bool, threshold int) (Transition, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Transition{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		status   string
		failures int
		since    time.Time
	)
	err = tx.QueryRow(ctx,
		`SELECT status, consecutive_failures, status_changed_at
		 FROM checks WHERE id = $1 FOR UPDATE`, id).Scan(&status, &failures, &since)
	if errors.Is(err, pgx.ErrNoRows) {
		return Transition{}, ErrNotFound
	}
	if err != nil {
		return Transition{}, err
	}

	next, failures := NextState(status, failures, success, threshold)
	_, err = tx.Exec(ctx,
		`UPDATE checks
		 SET status = $2, consecutive_failures = $3,
		     status_changed_at = CASE WHEN $4 THEN now() ELSE status_changed_at END
		 WHERE id = $1`, id, next, failures, next != status)
	if err != nil {
		return Transition{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Transition{}, err
	}
	return Transition{From: status, To: next, PrevSince: since}, nil
}

func (s *PostgresStore) CountByStatus(ctx context.Context) (map[string]int, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT status, count(*) FROM checks WHERE enabled GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var (
			status string
			n      int
		)
		if err := rows.Scan(&status, &n); err != nil {
			return nil, err
		}
		counts[status] = n
	}
	return counts, rows.Err()
}

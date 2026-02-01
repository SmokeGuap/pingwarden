package check

import (
	"context"
	"errors"

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

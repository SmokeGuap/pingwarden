package result

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ pool *pgxpool.Pool }

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool: pool} }

func (s *PostgresStore) Insert(ctx context.Context, r *Result) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO check_results (check_id, success, status_code, latency_ms, error)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id, checked_at`,
		r.CheckID, r.Success, r.StatusCode, r.LatencyMs, r.Error,
	).Scan(&r.ID, &r.CheckedAt)
}

func (s *PostgresStore) ListByCheck(ctx context.Context, checkID int64, limit int) ([]Result, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, check_id, checked_at, success, status_code, latency_ms, error
		 FROM check_results WHERE check_id = $1
		 ORDER BY checked_at DESC LIMIT $2`, checkID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Result])
}

func (s *PostgresStore) Prune(ctx context.Context, checkID int64, keep int) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM check_results
		WHERE check_id = $1
		  AND id < (
		      SELECT id FROM check_results
		      WHERE check_id = $1
		      ORDER BY id DESC
		      OFFSET $2 - 1 LIMIT 1
		  )`, checkID, keep)
	return err
}

package check

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("check not found")

type Check struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	URL             string    `json:"url"`
	IntervalSeconds int       `json:"interval_seconds"`
	Enabled         bool      `json:"enabled"`
	CreatedAt       time.Time `json:"created_at"`
}

type Store interface {
	Create(ctx context.Context, c *Check) error
	List(ctx context.Context) ([]Check, error)
	Get(ctx context.Context, id int64) (Check, error)
	Update(ctx context.Context, c *Check) error
	Delete(ctx context.Context, id int64) error
}

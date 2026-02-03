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

const (
	StatusUnknown = "unknown"
	StatusUp      = "up"
	StatusDown    = "down"
)

type Transition struct {
	From, To  string
	PrevSince time.Time
}

func (t Transition) Changed() bool { return t.From != t.To }

func NextState(status string, failures int, success bool, threshold int) (string, int) {
	if success {
		return StatusUp, 0
	}
	failures++
	if failures >= threshold {
		return StatusDown, failures
	}
	return status, failures
}

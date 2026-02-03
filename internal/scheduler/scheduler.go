package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/smokeguap/pingwarden/internal/metrics"
)

const batchSize = 500

type Claimer interface {
	ClaimDue(ctx context.Context, limit int) ([]int64, error)
}

type Publisher interface {
	Push(ctx context.Context, ids ...int64) error
}

type Scheduler struct {
	claimer Claimer
	pub     Publisher
	tick    time.Duration
	log     *slog.Logger
}

func New(c Claimer, p Publisher, tick time.Duration, log *slog.Logger) *Scheduler {
	return &Scheduler{claimer: c, pub: p, tick: tick, log: log}
}

func (s *Scheduler) Run(ctx context.Context) {
	t := time.NewTicker(s.tick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.enqueueDue(ctx)
		}
	}
}

func (s *Scheduler) enqueueDue(ctx context.Context) {
	for {
		ids, err := s.claimer.ClaimDue(ctx, batchSize)
		if err != nil {
			if ctx.Err() == nil {
				s.log.Error("claim due checks", "err", err)
			}
			return
		}
		if len(ids) == 0 {
			return
		}
		if err := s.pub.Push(ctx, ids...); err != nil {
			s.log.Error("push to queue", "err", err, "count", len(ids))
			return
		}
		metrics.JobsEnqueued.Add(float64(len(ids)))
		s.log.Debug("enqueued", "count", len(ids))
		if len(ids) < batchSize {
			return
		}
	}
}

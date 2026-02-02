package worker

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/smokeguap/pingwarden/internal/check"
	"github.com/smokeguap/pingwarden/internal/probe"
	"github.com/smokeguap/pingwarden/internal/queue"
	"github.com/smokeguap/pingwarden/internal/result"
)

type Queue interface {
	Pop(ctx context.Context, timeout time.Duration) (int64, error)
}

type Checks interface {
	Get(ctx context.Context, id int64) (check.Check, error)
}

type Results interface {
	Insert(ctx context.Context, r *result.Result) error
}

type Worker struct {
	queue       Queue
	checks      Checks
	results     Results
	client      *http.Client
	concurrency int
	timeout     time.Duration
	log         *slog.Logger
}

func New(q Queue, c Checks, r Results, concurrency int, timeout time.Duration, log *slog.Logger) *Worker {
	return &Worker{
		queue: q, checks: c, results: r,
		client:      &http.Client{},
		concurrency: concurrency,
		timeout:     timeout,
		log:         log,
	}
}

func (w *Worker) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for i := 0; i < w.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.loop(ctx)
		}()
	}
	wg.Wait()
}

func (w *Worker) loop(ctx context.Context) {
	for ctx.Err() == nil {
		id, err := w.queue.Pop(ctx, 2*time.Second)
		if errors.Is(err, queue.ErrEmpty) {
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			w.log.Error("pop job", "err", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}
		w.process(ctx, id)
	}
}

func (w *Worker) process(ctx context.Context, id int64) {
	ctx = context.WithoutCancel(ctx)

	c, err := w.checks.Get(ctx, id)
	switch {
	case errors.Is(err, check.ErrNotFound):
		return
	case err != nil:
		w.log.Error("get check", "id", id, "err", err)
		return
	case !c.Enabled:
		return
	}

	out := probe.HTTP(ctx, w.client, c.URL, w.timeout)
	res := result.Result{
		CheckID:    c.ID,
		Success:    out.Success,
		StatusCode: out.StatusCode,
		LatencyMs:  int(out.Latency.Milliseconds()),
		Error:      out.Err,
	}
	if err := w.results.Insert(ctx, &res); err != nil {
		w.log.Error("save result", "check_id", c.ID, "err", err)
		return
	}
	w.log.Info("check done",
		"check_id", c.ID, "url", c.URL,
		"success", res.Success, "latency_ms", res.LatencyMs)
}

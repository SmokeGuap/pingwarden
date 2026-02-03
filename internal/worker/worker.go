package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/smokeguap/pingwarden/internal/alert"
	"github.com/smokeguap/pingwarden/internal/check"
	"github.com/smokeguap/pingwarden/internal/metrics"
	"github.com/smokeguap/pingwarden/internal/probe"
	"github.com/smokeguap/pingwarden/internal/queue"
	"github.com/smokeguap/pingwarden/internal/result"
)

type Queue interface {
	Pop(ctx context.Context, timeout time.Duration) (int64, error)
}

type Events interface {
	PushEvent(ctx context.Context, payload []byte) error
}

type Checks interface {
	Get(ctx context.Context, id int64) (check.Check, error)
	RecordOutcome(ctx context.Context, id int64, success bool, threshold int) (check.Transition, error)
}

type Results interface {
	Insert(ctx context.Context, r *result.Result) error
}

type Config struct {
	Concurrency      int
	Timeout          time.Duration
	FailureThreshold int
}

type Worker struct {
	queue   Queue
	checks  Checks
	results Results
	events  Events
	client  *http.Client
	cfg     Config
	log     *slog.Logger
}

func New(q Queue, c Checks, r Results, e Events, cfg Config, log *slog.Logger) *Worker {
	return &Worker{
		queue: q, checks: c, results: r, events: e,
		client: &http.Client{},
		cfg:    cfg,
		log:    log,
	}
}

func (w *Worker) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for i := 0; i < w.cfg.Concurrency; i++ {
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

	out := probe.HTTP(ctx, w.client, c.URL, w.cfg.Timeout)

	outcome := "failure"
	if out.Success {
		outcome = "success"
	}
	metrics.ChecksTotal.WithLabelValues(outcome).Inc()
	metrics.CheckDuration.WithLabelValues(outcome).Observe(out.Latency.Seconds())

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

	tr, err := w.checks.RecordOutcome(ctx, c.ID, res.Success, w.cfg.FailureThreshold)
	if errors.Is(err, check.ErrNotFound) {
		return
	}
	if err != nil {
		w.log.Error("record outcome", "check_id", c.ID, "err", err)
		return
	}
	if tr.Changed() {
		metrics.StateTransitions.WithLabelValues(tr.To).Inc()
		w.log.Info("state changed", "check_id", c.ID, "from", tr.From, "to", tr.To)
	}
	if ev, ok := alert.FromTransition(c, tr, res); ok {
		w.publish(ctx, ev)
	}

	w.log.Info("check done",
		"check_id", c.ID, "url", c.URL,
		"success", res.Success, "latency_ms", res.LatencyMs)
}

func (w *Worker) publish(ctx context.Context, ev alert.Event) {
	payload, err := json.Marshal(ev)
	if err == nil {
		err = w.events.PushEvent(ctx, payload)
	}
	if err != nil {
		w.log.Error("publish alert", "check_id", ev.CheckID, "err", err)
		return
	}
	w.log.Info("alert queued", "check_id", ev.CheckID, "status", ev.Status)
}

package notifier

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/smokeguap/pingwarden/internal/alert"
	"github.com/smokeguap/pingwarden/internal/metrics"
	"github.com/smokeguap/pingwarden/internal/queue"
)

type Source interface {
	PopEvent(ctx context.Context, timeout time.Duration) ([]byte, error)
}

type Notifier struct {
	src      Source
	senders  []Sender
	attempts int
	backoff  time.Duration
	log      *slog.Logger
}

func New(src Source, senders []Sender, log *slog.Logger) *Notifier {
	return &Notifier{src: src, senders: senders, attempts: 4, backoff: time.Second, log: log}
}

func (n *Notifier) Run(ctx context.Context) {
	for ctx.Err() == nil {
		payload, err := n.src.PopEvent(ctx, 2*time.Second)
		if errors.Is(err, queue.ErrEmpty) {
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			n.log.Error("pop event", "err", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}

		var ev alert.Event
		if err := json.Unmarshal(payload, &ev); err != nil {
			n.log.Error("bad event payload", "err", err)
			continue
		}
		n.dispatch(ctx, ev)
	}
}

func (n *Notifier) dispatch(ctx context.Context, ev alert.Event) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	for _, s := range n.senders {
		n.sendWithRetry(ctx, s, ev)
	}
}

func (n *Notifier) sendWithRetry(ctx context.Context, s Sender, ev alert.Event) {
	delay := n.backoff
	for attempt := 1; ; attempt++ {
		err := s.Send(ctx, ev)
		if err == nil {
			metrics.Notifications.WithLabelValues(s.Name(), "ok").Inc()
			n.log.Info("notification sent", "channel", s.Name(), "check_id", ev.CheckID, "status", ev.Status)
			return
		}
		if attempt >= n.attempts || ctx.Err() != nil {
			metrics.Notifications.WithLabelValues(s.Name(), "failed").Inc()
			n.log.Error("notification dropped", "channel", s.Name(), "check_id", ev.CheckID, "err", err)
			return
		}
		n.log.Warn("send failed, retrying", "channel", s.Name(), "attempt", attempt, "err", err)
		select {
		case <-ctx.Done():
		case <-time.After(delay):
		}
		delay *= 2 // 1s, 2s, 4s
	}
}

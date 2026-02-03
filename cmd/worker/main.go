package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/smokeguap/pingwarden/internal/check"
	"github.com/smokeguap/pingwarden/internal/config"
	"github.com/smokeguap/pingwarden/internal/db"
	"github.com/smokeguap/pingwarden/internal/metrics"
	"github.com/smokeguap/pingwarden/internal/queue"
	"github.com/smokeguap/pingwarden/internal/result"
	"github.com/smokeguap/pingwarden/internal/worker"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	q := queue.New(cfg.RedisAddr)
	defer q.Close()
	if err := q.Ping(ctx); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}

	go metrics.Serve(ctx, cfg.MetricsAddr, log)

	w := worker.New(q, check.NewPostgresStore(pool), result.NewPostgresStore(pool), q,
		worker.Config{
			Concurrency:      cfg.WorkerConcurrency,
			Timeout:          cfg.CheckTimeout,
			FailureThreshold: cfg.AlertFailureThreshold,
		}, log)

	log.Info("worker started", "concurrency", cfg.WorkerConcurrency)
	w.Run(ctx)
	log.Info("worker stopped")
	return nil
}

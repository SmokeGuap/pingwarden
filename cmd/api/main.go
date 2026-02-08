package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/smokeguap/pingwarden/internal/auth"
	"github.com/smokeguap/pingwarden/internal/check"
	"github.com/smokeguap/pingwarden/internal/config"
	"github.com/smokeguap/pingwarden/internal/db"
	"github.com/smokeguap/pingwarden/internal/metrics"
	"github.com/smokeguap/pingwarden/internal/result"
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
	if err := cfg.RequireDB(); err != nil {
		return err
	}
	if err := cfg.RequireAPIToken(); err != nil {
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

	checkStore := check.NewPostgresStore(pool)
	metrics.RegisterChecksGauge(checkStore)

	go metrics.Serve(ctx, cfg.MetricsAddr, log)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		pctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(pctx); err != nil {
			http.Error(w, "db unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	check.NewHandler(checkStore, log).Register(mux)
	result.NewHandler(result.NewPostgresStore(pool), log).Register(mux)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           metrics.Middleware(auth.Bearer(cfg.APIToken, mux)),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("server started", "addr", cfg.HTTPAddr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		log.Info("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

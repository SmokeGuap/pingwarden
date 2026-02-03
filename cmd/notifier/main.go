package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/smokeguap/pingwarden/internal/config"
	"github.com/smokeguap/pingwarden/internal/metrics"
	"github.com/smokeguap/pingwarden/internal/notifier"
	"github.com/smokeguap/pingwarden/internal/queue"
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

	var senders []notifier.Sender
	if cfg.TelegramToken != "" && cfg.TelegramChatID != "" {
		senders = append(senders, notifier.NewTelegram(cfg.TelegramToken, cfg.TelegramChatID))
	}
	if cfg.WebhookURL != "" {
		senders = append(senders, notifier.NewWebhook(cfg.WebhookURL))
	}
	if len(senders) == 0 {
		return errors.New("no channels configured: set TELEGRAM_BOT_TOKEN + TELEGRAM_CHAT_ID and/or ALERT_WEBHOOK_URL")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	q := queue.New(cfg.RedisAddr)
	defer q.Close()
	if err := q.Ping(ctx); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}

	go metrics.Serve(ctx, cfg.MetricsAddr, log)

	log.Info("notifier started", "channels", len(senders))
	notifier.New(q, senders, log).Run(ctx)
	log.Info("notifier stopped")
	return nil
}

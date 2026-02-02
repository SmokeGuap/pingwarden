package main

import (
	"log/slog"
	"os"

	"github.com/smokeguap/pingwarden/internal/config"
	"github.com/smokeguap/pingwarden/internal/db"
)

func main() {
	cfg, err := config.Load()
	if err == nil {
		err = db.Migrate(cfg.DatabaseURL)
	}
	if err != nil {
		slog.Error("migrate failed", "err", err)
		os.Exit(1)
	}
	slog.Info("migrations applied")
}

package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr              string
	DatabaseURL           string
	RedisAddr             string
	WorkerConcurrency     int
	SchedulerTick         time.Duration
	CheckTimeout          time.Duration
	MetricsAddr           string
	AlertFailureThreshold int
	TelegramToken         string
	TelegramChatID        string
	WebhookURL            string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:       getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		RedisAddr:      getEnv("REDIS_ADDR", "localhost:6379"),
		MetricsAddr:    getEnv("METRICS_ADDR", ":9090"),
		TelegramToken:  os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChatID: os.Getenv("TELEGRAM_CHAT_ID"),
		WebhookURL:     os.Getenv("ALERT_WEBHOOK_URL"),
	}

	var err error
	if cfg.WorkerConcurrency, err = getInt("WORKER_CONCURRENCY", 10); err != nil {
		return Config{}, err
	}
	if cfg.AlertFailureThreshold, err = getInt("ALERT_FAILURE_THRESHOLD", 3); err != nil {
		return Config{}, err
	}
	if cfg.SchedulerTick, err = getDuration("SCHEDULER_TICK", time.Second); err != nil {
		return Config{}, err
	}
	if cfg.CheckTimeout, err = getDuration("CHECK_TIMEOUT", 10*time.Second); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) RequireDB() error {
	if c.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	return nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer, got %q", key, v)
	}
	return n, nil
}

func getDuration(key string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration (e.g. 5s), got %q", key, v)
	}
	return d, nil
}

package probe

import (
	"context"
	"io"
	"net/http"
	"time"
)

type Outcome struct {
	Success    bool
	StatusCode *int
	Latency    time.Duration
	Err        string
}

func HTTP(ctx context.Context, client *http.Client, url string, timeout time.Duration) Outcome {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Outcome{Err: err.Error()}
	}
	req.Header.Set("User-Agent", "pingwarden/0.1")

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start)
	if err != nil {
		return Outcome{Latency: latency, Err: err.Error()}
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))

	code := resp.StatusCode
	return Outcome{Success: code < 400, StatusCode: &code, Latency: latency}
}

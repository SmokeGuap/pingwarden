package result

import "time"

type Result struct {
	ID         int64     `json:"id"`
	CheckID    int64     `json:"check_id"`
	CheckedAt  time.Time `json:"checked_at"`
	Success    bool      `json:"success"`
	StatusCode *int      `json:"status_code"`
	LatencyMs  int       `json:"latency_ms"`
	Error      string    `json:"error"`
}

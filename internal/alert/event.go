package alert

import (
	"fmt"
	"time"

	"github.com/smokeguap/pingwarden/internal/check"
	"github.com/smokeguap/pingwarden/internal/result"
)

type Event struct {
	CheckID        int64     `json:"check_id"`
	Name           string    `json:"name"`
	URL            string    `json:"url"`
	Status         string    `json:"status"`
	StatusCode     *int      `json:"status_code,omitempty"`
	Error          string    `json:"error,omitempty"`
	At             time.Time `json:"at"`
	DownForSeconds int64     `json:"down_for_seconds,omitempty"`
}

func FromTransition(c check.Check, t check.Transition, r result.Result) (Event, bool) {
	ev := Event{CheckID: c.ID, Name: c.Name, URL: c.URL, At: r.CheckedAt}
	switch {
	case t.To == check.StatusDown && t.From != check.StatusDown:
		ev.Status = check.StatusDown
		ev.StatusCode = r.StatusCode
		ev.Error = r.Error
		return ev, true
	case t.To == check.StatusUp && t.From == check.StatusDown:
		ev.Status = check.StatusUp
		ev.DownForSeconds = int64(r.CheckedAt.Sub(t.PrevSince).Seconds())
		return ev, true
	}
	return Event{}, false
}

func (e Event) Text() string {
	if e.Status == check.StatusUp {
		return fmt.Sprintf("RECOVERED: %s\n%s\nDowntime: %s",
			e.Name, e.URL, time.Duration(e.DownForSeconds)*time.Second)
	}
	reason := e.Error
	if reason == "" && e.StatusCode != nil {
		reason = fmt.Sprintf("HTTP %d", *e.StatusCode)
	}
	return fmt.Sprintf("DOWN: %s\n%s\nReason: %s", e.Name, e.URL, reason)
}

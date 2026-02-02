package check

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeStore struct {
	items map[int64]Check
	next  int64
}

func newFakeStore() *fakeStore { return &fakeStore{items: map[int64]Check{}} }

func (f *fakeStore) Create(_ context.Context, c *Check) error {
	f.next++
	c.ID, c.CreatedAt = f.next, time.Now()
	f.items[c.ID] = *c
	return nil
}
func (f *fakeStore) List(context.Context) ([]Check, error) { return nil, nil }
func (f *fakeStore) Get(_ context.Context, id int64) (Check, error) {
	c, ok := f.items[id]
	if !ok {
		return Check{}, ErrNotFound
	}
	return c, nil
}
func (f *fakeStore) Update(_ context.Context, c *Check) error {
	if _, ok := f.items[c.ID]; !ok {
		return ErrNotFound
	}
	f.items[c.ID] = *c
	return nil
}
func (f *fakeStore) Delete(_ context.Context, id int64) error {
	if _, ok := f.items[id]; !ok {
		return ErrNotFound
	}
	delete(f.items, id)
	return nil
}

func newServer() http.Handler {
	mux := http.NewServeMux()
	NewHandler(newFakeStore(), slog.New(slog.NewTextHandler(io.Discard, nil))).Register(mux)
	return mux
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{"ok", `{"name":"site","url":"https://example.com","interval_seconds":30}`, http.StatusCreated},
		{"defaults", `{"name":"site","url":"https://example.com"}`, http.StatusCreated},
		{"empty name", `{"name":"","url":"https://example.com"}`, http.StatusBadRequest},
		{"bad url", `{"name":"x","url":"ftp://nope"}`, http.StatusBadRequest},
		{"interval too small", `{"name":"x","url":"https://a.io","interval_seconds":1}`, http.StatusBadRequest},
		{"unknown field", `{"name":"x","url":"https://a.io","foo":1}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(newServer(), http.MethodPost, "/api/v1/checks", tt.body)
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d, body: %s", rec.Code, tt.want, rec.Body)
			}
		})
	}
}

func TestGetNotFound(t *testing.T) {
	rec := do(newServer(), http.MethodGet, "/api/v1/checks/42", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

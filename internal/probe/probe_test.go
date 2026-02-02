package probe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fail":
			w.WriteHeader(http.StatusInternalServerError)
		case "/slow":
			time.Sleep(300 * time.Millisecond)
		}
	}))
	defer srv.Close()

	tests := []struct {
		name        string
		path        string
		timeout     time.Duration
		wantSuccess bool
		wantErr     bool
	}{
		{"ok", "/ok", time.Second, true, false},
		{"server error", "/fail", time.Second, false, false},
		{"timeout", "/slow", 50 * time.Millisecond, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := HTTP(context.Background(), srv.Client(), srv.URL+tt.path, tt.timeout)
			if out.Success != tt.wantSuccess {
				t.Fatalf("success = %v, want %v", out.Success, tt.wantSuccess)
			}
			if (out.Err != "") != tt.wantErr {
				t.Fatalf("err = %q, wantErr %v", out.Err, tt.wantErr)
			}
		})
	}
}

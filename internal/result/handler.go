package result

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

type Lister interface {
	ListByCheck(ctx context.Context, checkID int64, limit int) ([]Result, error)
}

type Handler struct {
	store Lister
	log   *slog.Logger
}

func NewHandler(store Lister, log *slog.Logger) *Handler {
	return &Handler{store: store, log: log}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/checks/{id}/results", h.list)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 || n > 500 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "limit must be 1..500"})
			return
		}
		limit = n
	}
	results, err := h.store.ListByCheck(r.Context(), id, limit)
	if err != nil {
		h.log.Error("list results", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if results == nil {
		results = []Result{}
	}
	writeJSON(w, http.StatusOK, results)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

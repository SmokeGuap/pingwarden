package check

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Handler struct {
	store Store
	log   *slog.Logger
}

func NewHandler(store Store, log *slog.Logger) *Handler {
	return &Handler{store: store, log: log}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/checks", h.create)
	mux.HandleFunc("GET /api/v1/checks", h.list)
	mux.HandleFunc("GET /api/v1/checks/{id}", h.get)
	mux.HandleFunc("PUT /api/v1/checks/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/checks/{id}", h.delete)
}

type input struct {
	Name            string `json:"name"`
	URL             string `json:"url"`
	IntervalSeconds int    `json:"interval_seconds"`
	Enabled         *bool  `json:"enabled"`
}

func (in input) toCheck() (Check, error) {
	if strings.TrimSpace(in.Name) == "" {
		return Check{}, errors.New("name is required")
	}
	u, err := url.ParseRequestURI(in.URL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return Check{}, errors.New("url must be a valid http(s) URL")
	}
	interval := in.IntervalSeconds
	if interval == 0 {
		interval = 60
	}
	if interval < 10 {
		return Check{}, errors.New("interval_seconds must be >= 10")
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	return Check{Name: in.Name, URL: in.URL, IntervalSeconds: interval, Enabled: enabled}, nil
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	c, ok := h.decode(w, r)
	if !ok {
		return
	}
	if err := h.store.Create(r.Context(), &c); err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	checks, err := h.store.List(r.Context())
	if err != nil {
		h.fail(w, err)
		return
	}
	if checks == nil {
		checks = []Check{}
	}
	writeJSON(w, http.StatusOK, checks)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	c, err := h.store.Get(r.Context(), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	c, ok := h.decode(w, r)
	if !ok {
		return
	}
	c.ID = id
	if err := h.store.Update(r.Context(), &c); err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) decode(w http.ResponseWriter, r *http.Request) (Check, bool) {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	var in input
	if err := dec.Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return Check{}, false
	}
	c, err := in.toCheck()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return Check{}, false
	}
	return c, true
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "check not found")
		return
	}
	h.log.Error("store error", "err", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

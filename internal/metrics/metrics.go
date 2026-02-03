package metrics

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	HTTPRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pingwarden_http_requests_total",
		Help: "HTTP requests handled by the API.",
	}, []string{"route", "status"})

	HTTPDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "pingwarden_http_request_duration_seconds",
		Help:    "API request latency.",
		Buckets: prometheus.DefBuckets,
	}, []string{"route"})

	ChecksTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pingwarden_checks_total",
		Help: "Executed checks by result.",
	}, []string{"result"})

	CheckDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "pingwarden_check_duration_seconds",
		Help:    "Latency of monitored targets as seen by workers.",
		Buckets: []float64{.05, .1, .25, .5, 1, 2.5, 5, 10},
	}, []string{"result"})

	StateTransitions = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pingwarden_state_transitions_total",
		Help: "Check state changes by new state.",
	}, []string{"to"})

	JobsEnqueued = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pingwarden_jobs_enqueued_total",
		Help: "Jobs put into the queue by the scheduler.",
	})

	Notifications = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pingwarden_notifications_total",
		Help: "Notification delivery attempts by channel and outcome.",
	}, []string{"channel", "outcome"})
)

func init() {
	for _, r := range []string{"success", "failure"} {
		ChecksTotal.WithLabelValues(r)
		CheckDuration.WithLabelValues(r)
	}
}

func Handler() http.Handler { return promhttp.Handler() }

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, r)

		route := r.Pattern
		if route == "" {
			route = "unmatched"
		}
		HTTPRequests.WithLabelValues(route, strconv.Itoa(rec.status)).Inc()
		HTTPDuration.WithLabelValues(route).Observe(time.Since(start).Seconds())
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

func Serve(ctx context.Context, addr string, log *slog.Logger) {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", Handler())
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}

	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}()

	log.Info("metrics server started", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("metrics server", "err", err)
	}
}

type StatusCounter interface {
	CountByStatus(ctx context.Context) (map[string]int, error)
}

func RegisterChecksGauge(src StatusCounter) {
	prometheus.MustRegister(&checksCollector{
		src: src,
		desc: prometheus.NewDesc("pingwarden_checks",
			"Enabled checks by current state.", []string{"status"}, nil),
	})
}

type checksCollector struct {
	src  StatusCounter
	desc *prometheus.Desc
}

func (c *checksCollector) Describe(ch chan<- *prometheus.Desc) { ch <- c.desc }

func (c *checksCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	counts, err := c.src.CountByStatus(ctx)
	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.desc, err)
		return
	}
	for _, s := range []string{"up", "down", "unknown"} {
		ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, float64(counts[s]), s)
	}
}

package api_middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// httpRequestsTotal is a counter for total HTTP requests.
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of http requests.",
		},
		[]string{"method", "path", "status"},
	)

	// httpRequestDuration is a histogram for request latencies.
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "Duration of HTTP requests.",
		},
		[]string{"method", "path"},
	)
)

// Telemetry is a middleware that records Prometheus metrics and logs requests.
func Telemetry(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Start a timer
		startTime := time.Now()

		// Use a response writer wrapper to capture the status code
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		// Call the next handler in the chain
		next.ServeHTTP(ww, r)

		// Record the duration and increment the request counter
		duration := time.Since(startTime)
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration.Seconds())
		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, http.StatusText(ww.Status())).Inc()

		// Log the request details using slog
		slog.Info("request handled",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"duration", duration,
			"user_agent", r.UserAgent(),
		)
	})
}

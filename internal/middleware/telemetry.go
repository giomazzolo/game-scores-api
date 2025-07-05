package api_middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
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
		startTime := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		// Call the next handler in the chain
		next.ServeHTTP(ww, r)

		// Get the route pattern from the chi context.
		routePattern := chi.RouteContext(r.Context()).RoutePattern()
		if routePattern == "" {
			// If no route was matched, use the raw path.
			// This can happen for 404s.
			routePattern = r.URL.Path
		}

		// Record metrics with the clean route pattern.
		duration := time.Since(startTime)
		httpRequestDuration.WithLabelValues(r.Method, routePattern).Observe(duration.Seconds())
		httpRequestsTotal.WithLabelValues(r.Method, routePattern, http.StatusText(ww.Status())).Inc()

		// Log the request details using slog
		slog.Info("request handled",
			"method", r.Method,
			"path", routePattern,
			"status", ww.Status(),
			"duration", duration,
			"user_agent", r.UserAgent(),
		)
	})
}

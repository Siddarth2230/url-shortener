package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Siddarth2230/url-shortener/pkg/metrics"
)

// MetricsMiddleware tracks HTTP request metrics
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Handle request
		next.ServeHTTP(ww, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(ww.statusCode)

		metrics.RequestDuration.WithLabelValues(r.Method, status).Observe(duration)
		metrics.RequestTotal.WithLabelValues(r.Method, status).Inc()
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

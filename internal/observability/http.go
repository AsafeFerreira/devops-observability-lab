package observability

import (
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

var safeTenantValue = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,31}$`)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (writer *statusWriter) WriteHeader(status int) {
	writer.status = status
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *statusWriter) Write(body []byte) (int, error) {
	if writer.status == 0 {
		writer.WriteHeader(http.StatusOK)
	}
	return writer.ResponseWriter.Write(body)
}

func Correlation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		correlationID := SanitizeCorrelationID(request.Header.Get("X-Correlation-ID"))
		ctx := WithCorrelationID(request.Context(), correlationID)
		if tenant := normalizeTenantForLogs(request.Header.Get("X-Tenant-ID")); tenant != "" {
			ctx = WithTenantID(ctx, tenant)
		}
		writer.Header().Set("X-Correlation-ID", correlationID)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

func normalizeTenantForLogs(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if safeTenantValue.MatchString(value) {
		return value
	}
	return ""
}

func HTTPMetrics(metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			started := time.Now()
			wrapped := &statusWriter{ResponseWriter: writer, status: http.StatusOK}
			next.ServeHTTP(wrapped, request)

			route := chi.RouteContext(request.Context()).RoutePattern()
			if route == "" {
				route = "unmatched"
			}
			status := strconv.Itoa(wrapped.status)
			metrics.HTTPRequests.WithLabelValues(route, request.Method, status).Inc()
			observation := metrics.HTTPDuration.WithLabelValues(route, request.Method)
			duration := time.Since(started)
			spanContext := trace.SpanContextFromContext(request.Context())
			if exemplar, ok := observation.(prometheus.ExemplarObserver); spanContext.IsValid() && ok {
				exemplar.ObserveWithExemplar(duration.Seconds(), prometheus.Labels{
					"trace_id": spanContext.TraceID().String(),
				})
			} else {
				observation.Observe(duration.Seconds())
			}

			attributes := append(LogAttributes(request.Context()),
				"operation", "http_request",
				"method", request.Method,
				"route", route,
				"status", wrapped.status,
				"duration_ms", duration.Milliseconds(),
			)
			switch {
			case wrapped.status >= 500:
				slog.ErrorContext(request.Context(), "http request completed", attributes...)
			case wrapped.status >= 400:
				slog.WarnContext(request.Context(), "http request completed", attributes...)
			case route != "/health/live" && route != "/health/ready" && route != "/metrics":
				slog.InfoContext(request.Context(), "http request completed", attributes...)
			}
		})
	}
}

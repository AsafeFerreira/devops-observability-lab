package simulator

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/asafeferreira/devops-observability-lab/internal/observability"
)

type Handler struct {
	labMode  bool
	metrics  *observability.Metrics
	requests atomic.Uint64
}

type ProcessRequest struct {
	ImportID    string `json:"importId"`
	Source      string `json:"source"`
	RecordCount int    `json:"recordCount"`
}

func New(labMode bool, metrics *observability.Metrics) http.Handler {
	handler := &Handler{labMode: labMode, metrics: metrics}
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)
	router.Use(observability.Correlation)
	router.Use(observability.HTTPMetrics(metrics))
	router.Get("/health/live", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]string{"status": "alive"})
	})
	router.Get("/health/ready", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ready"})
	})
	router.Handle("/metrics", promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{EnableOpenMetrics: true}))
	router.Post("/api/v1/process", handler.process)
	return router
}

func (handler *Handler) process(writer http.ResponseWriter, request *http.Request) {
	ctx, span := otel.Tracer("lab/integration-simulator").Start(request.Context(), "simulate external processing")
	defer span.End()

	var body ProcessRequest
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, 32*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil || body.ImportID == "" || body.RecordCount < 1 {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	span.SetAttributes(
		attribute.String("lab.import.source", body.Source),
		attribute.Int("lab.import.record_count", body.RecordCount),
	)

	sequence := handler.requests.Add(1)
	if handler.labMode {
		switch body.Source {
		case "slow":
			timer := time.NewTimer(800 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		case "force-error":
			span.SetStatus(codes.Error, "forced lab failure")
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "forced integration failure"})
			return
		case "flaky":
			if sequence%2 == 1 {
				span.SetStatus(codes.Error, "intermittent lab failure")
				writeJSON(writer, http.StatusServiceUnavailable, map[string]string{"error": "intermittent integration failure"})
				return
			}
		}
	}

	writeJSON(writer, http.StatusOK, map[string]any{
		"status":           "processed",
		"importId":         body.ImportID,
		"processedRecords": body.RecordCount,
	})
}

func writeJSON(writer http.ResponseWriter, status int, body any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(body)
}

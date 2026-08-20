package processor

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/asafeferreira/devops-observability-lab/internal/messaging"
	"github.com/asafeferreira/devops-observability-lab/internal/observability"
)

type HealthRepository interface {
	Ping(context.Context) error
}

func NewHealthHandler(repository HealthRepository, broker *messaging.Broker, metrics *observability.Metrics) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)
	router.Use(observability.Correlation)
	router.Use(observability.HTTPMetrics(metrics))
	router.Get("/health/live", func(writer http.ResponseWriter, _ *http.Request) {
		writeHealth(writer, http.StatusOK, "alive")
	})
	router.Get("/health/ready", func(writer http.ResponseWriter, request *http.Request) {
		if err := repository.Ping(request.Context()); err != nil || broker == nil || !broker.Healthy() {
			writeHealth(writer, http.StatusServiceUnavailable, "unavailable")
			return
		}
		writeHealth(writer, http.StatusOK, "ready")
	})
	router.Handle("/metrics", promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{EnableOpenMetrics: true}))
	return router
}

func writeHealth(writer http.ResponseWriter, status int, state string) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(map[string]string{"status": state})
}

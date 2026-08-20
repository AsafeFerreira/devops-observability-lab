package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/asafeferreira/devops-observability-lab/internal/database"
	"github.com/asafeferreira/devops-observability-lab/internal/messaging"
	"github.com/asafeferreira/devops-observability-lab/internal/model"
	"github.com/asafeferreira/devops-observability-lab/internal/observability"
)

type ImportRepository interface {
	CreateImport(context.Context, string, string, model.CreateImportRequest, string, map[string]string) (*model.Import, error)
	GetImport(context.Context, string, string) (*model.Import, error)
	ListImports(context.Context, string, int) ([]model.Import, error)
	Ping(context.Context) error
}

type API struct {
	repository ImportRepository
	broker     *messaging.Broker
	metrics    *observability.Metrics
}

var idempotencyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`)

func New(repository ImportRepository, broker *messaging.Broker, metrics *observability.Metrics) http.Handler {
	api := &API{repository: repository, broker: broker, metrics: metrics}
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)
	router.Use(observability.Correlation)
	router.Use(observability.HTTPMetrics(metrics))

	router.Get("/", api.info)
	router.Get("/health/live", api.live)
	router.Get("/health/ready", api.ready)
	router.Handle("/metrics", promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{EnableOpenMetrics: true}))
	router.Route("/api/v1/imports", func(router chi.Router) {
		router.Post("/", api.createImport)
		router.Get("/", api.listImports)
		router.Get("/{id}", api.getImport)
	})
	return router
}

func (api *API) info(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{
		"service": "imports-api",
		"status":  "ok",
	})
}

func (api *API) live(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"status": "alive"})
}

func (api *API) ready(writer http.ResponseWriter, request *http.Request) {
	if err := api.repository.Ping(request.Context()); err != nil {
		writeError(writer, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "database is unavailable", nil)
		return
	}
	if api.broker == nil || !api.broker.Healthy() {
		writeError(writer, http.StatusServiceUnavailable, "MESSAGING_UNAVAILABLE", "message broker is unavailable", nil)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ready"})
}

func (api *API) createImport(writer http.ResponseWriter, request *http.Request) {
	tenantID, ok := requiredTenant(writer, request)
	if !ok {
		return
	}
	idempotencyKey := strings.TrimSpace(request.Header.Get("Idempotency-Key"))
	if !idempotencyPattern.MatchString(idempotencyKey) {
		writeError(writer, http.StatusBadRequest, "INVALID_IDEMPOTENCY_KEY",
			"Idempotency-Key must have 8-128 safe characters", nil)
		return
	}

	var body model.CreateImportRequest
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, 32*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		writeError(writer, http.StatusBadRequest, "INVALID_JSON", "request body must be valid JSON", nil)
		return
	}
	if err := body.Validate(); err != nil {
		var validation model.ValidationError
		if errors.As(err, &validation) {
			writeError(writer, http.StatusBadRequest, "VALIDATION_ERROR", "request validation failed", validation.Fields)
			return
		}
		writeError(writer, http.StatusBadRequest, "VALIDATION_ERROR", "request validation failed", nil)
		return
	}

	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(request.Context(), carrier)
	created, err := api.repository.CreateImport(
		request.Context(), tenantID, idempotencyKey, body,
		observability.CorrelationID(request.Context()), map[string]string(carrier),
	)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create import", nil)
		return
	}

	status := http.StatusCreated
	if created.IdempotentReplay {
		status = http.StatusOK
		writer.Header().Set("Idempotent-Replay", "true")
	}
	writer.Header().Set("Location", "/api/v1/imports/"+created.ID)
	api.metrics.BusinessOperations.WithLabelValues("import_requested", "success").Inc()
	writeJSON(writer, status, created)
}

func (api *API) getImport(writer http.ResponseWriter, request *http.Request) {
	tenantID, ok := requiredTenant(writer, request)
	if !ok {
		return
	}
	item, err := api.repository.GetImport(request.Context(), tenantID, chi.URLParam(request, "id"))
	if errors.Is(err, database.ErrNotFound) {
		writeError(writer, http.StatusNotFound, "IMPORT_NOT_FOUND", "import was not found", nil)
		return
	}
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "INTERNAL_ERROR", "could not read import", nil)
		return
	}
	writeJSON(writer, http.StatusOK, item)
}

func (api *API) listImports(writer http.ResponseWriter, request *http.Request) {
	tenantID, ok := requiredTenant(writer, request)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(request.URL.Query().Get("limit"))
	items, err := api.repository.ListImports(request.Context(), tenantID, limit)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list imports", nil)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"items": items, "count": len(items)})
}

func requiredTenant(writer http.ResponseWriter, request *http.Request) (string, bool) {
	tenantID, err := model.NormalizeTenant(request.Header.Get("X-Tenant-ID"))
	if err != nil {
		writeError(writer, http.StatusBadRequest, "INVALID_TENANT", err.Error(), nil)
		return "", false
	}
	request.Header.Set("X-Tenant-ID", tenantID)
	return tenantID, true
}

type errorResponse struct {
	Code          string            `json:"code"`
	Message       string            `json:"message"`
	Fields        map[string]string `json:"fields,omitempty"`
	CorrelationID string            `json:"correlationId,omitempty"`
}

func writeError(writer http.ResponseWriter, status int, code, message string, fields map[string]string) {
	writeJSON(writer, status, errorResponse{Code: code, Message: message, Fields: fields})
}

func writeJSON(writer http.ResponseWriter, status int, body any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(writer).Encode(body)
	}
}

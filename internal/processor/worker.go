package processor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/asafeferreira/devops-observability-lab/internal/database"
	"github.com/asafeferreira/devops-observability-lab/internal/messaging"
	"github.com/asafeferreira/devops-observability-lab/internal/model"
	"github.com/asafeferreira/devops-observability-lab/internal/observability"
)

type Worker struct {
	store       *database.Store
	broker      *messaging.Broker
	integration *IntegrationClient
	metrics     *observability.Metrics
}

func NewWorker(store *database.Store, broker *messaging.Broker, integration *IntegrationClient, metrics *observability.Metrics) *Worker {
	return &Worker{store: store, broker: broker, integration: integration, metrics: metrics}
}

func (worker *Worker) Run(ctx context.Context) error {
	deliveries, err := worker.broker.Consume(ctx)
	if err != nil {
		return fmt.Errorf("start RabbitMQ consumer: %w", err)
	}
	heartbeat := time.NewTicker(10 * time.Second)
	counts := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	defer counts.Stop()
	worker.metrics.WorkerHeartbeat.SetToCurrentTime()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-heartbeat.C:
			worker.metrics.WorkerHeartbeat.SetToCurrentTime()
		case <-counts.C:
			worker.updateImportMetrics(ctx)
		case delivery, ok := <-deliveries:
			if !ok {
				if ctx.Err() != nil {
					return nil
				}
				return errors.New("RabbitMQ delivery stream closed")
			}
			worker.processDelivery(ctx, delivery)
		}
	}
}

func (worker *Worker) processDelivery(baseContext context.Context, delivery amqp.Delivery) {
	ctx := messaging.ExtractDeliveryContext(baseContext, delivery)
	ctx, span := otel.Tracer("lab/worker").Start(ctx, "rabbitmq consume import.requested",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "rabbitmq"),
			attribute.String("messaging.destination.name", messaging.QueueName),
			attribute.String("messaging.operation.name", "process"),
		),
	)
	defer span.End()

	var event model.ImportRequestedEvent
	if err := json.Unmarshal(delivery.Body, &event); err != nil {
		worker.failDelivery(ctx, delivery, "invalid event payload", err)
		return
	}
	ctx = observability.WithCorrelationID(ctx, observability.SanitizeCorrelationID(event.CorrelationID))
	ctx = observability.WithTenantID(ctx, event.TenantID)
	span.SetAttributes(
		attribute.String("lab.import.id", event.ImportID),
		attribute.String("lab.import.source", event.Source),
	)

	started := time.Now()
	item, err := worker.store.StartProcessing(ctx, event.ImportID)
	if err != nil {
		worker.failDelivery(ctx, delivery, "could not start import", err)
		return
	}
	err = worker.integration.Process(ctx, *item)
	if err != nil {
		safeError := safeMessage(err)
		if updateError := worker.store.CompleteImport(ctx, event.ImportID, model.StatusDeadLetter, safeError); updateError != nil {
			slog.ErrorContext(ctx, "could not persist failed import",
				append(observability.LogAttributes(ctx), "operation", "complete_import", "error", updateError)...)
		}
		worker.metrics.BusinessOperations.WithLabelValues("import_processed", "failure").Inc()
		worker.metrics.BusinessDuration.WithLabelValues("import_processed", "failure").Observe(time.Since(started).Seconds())
		span.RecordError(err)
		span.SetStatus(codes.Error, "import processing failed")
		worker.failDelivery(ctx, delivery, "import sent to dead-letter queue", err)
		return
	}

	if err := worker.store.CompleteImport(ctx, event.ImportID, model.StatusSucceeded, ""); err != nil {
		worker.failDelivery(ctx, delivery, "could not complete import", err)
		return
	}
	if err := delivery.Ack(false); err != nil {
		slog.ErrorContext(ctx, "could not acknowledge processed message",
			append(observability.LogAttributes(ctx), "operation", "queue_ack", "error", err)...)
		return
	}
	worker.metrics.QueueMessages.WithLabelValues("consume", "success").Inc()
	worker.metrics.BusinessOperations.WithLabelValues("import_processed", "success").Inc()
	worker.metrics.BusinessDuration.WithLabelValues("import_processed", "success").Observe(time.Since(started).Seconds())
	worker.metrics.WorkerLastSuccess.SetToCurrentTime()
	worker.updateImportMetrics(ctx)
	slog.InfoContext(ctx, "import processed",
		append(observability.LogAttributes(ctx),
			"operation", "import_processed",
			"import_id", item.ID,
			"source", item.Source,
			"record_count", item.RecordCount,
			"duration_ms", time.Since(started).Milliseconds(),
		)...,
	)
}

func (worker *Worker) failDelivery(ctx context.Context, delivery amqp.Delivery, message string, err error) {
	worker.metrics.QueueMessages.WithLabelValues("consume", "failure").Inc()
	slog.ErrorContext(ctx, message,
		append(observability.LogAttributes(ctx),
			"operation", "queue_consume",
			"error", safeMessage(err),
		)...,
	)
	if nackError := delivery.Nack(false, false); nackError != nil {
		slog.ErrorContext(ctx, "could not reject message", "operation", "queue_nack", "error", nackError)
	}
}

func (worker *Worker) updateImportMetrics(ctx context.Context) {
	counts, err := worker.store.CountImportsByStatus(ctx)
	if err != nil {
		slog.WarnContext(ctx, "could not update import status metrics", "operation", "status_metrics", "error", err)
		return
	}
	for _, status := range []model.ImportStatus{
		model.StatusQueued, model.StatusProcessing, model.StatusSucceeded, model.StatusFailed, model.StatusDeadLetter,
	} {
		worker.metrics.ImportsCurrent.WithLabelValues(string(status)).Set(float64(counts[status]))
	}
}

func safeMessage(err error) string {
	if err == nil {
		return ""
	}
	message := strings.ReplaceAll(err.Error(), "\n", " ")
	if len(message) > 300 {
		message = message[:300]
	}
	return message
}

package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/asafeferreira/devops-observability-lab/internal/database"
	"github.com/asafeferreira/devops-observability-lab/internal/model"
	"github.com/asafeferreira/devops-observability-lab/internal/observability"
)

type OutboxPublisher struct {
	store    *database.Store
	broker   *Broker
	interval time.Duration
	workerID string
}

func NewOutboxPublisher(store *database.Store, broker *Broker, interval time.Duration) *OutboxPublisher {
	hostname, _ := os.Hostname()
	return &OutboxPublisher{
		store:    store,
		broker:   broker,
		interval: interval,
		workerID: fmt.Sprintf("%s-%d", hostname, os.Getpid()),
	}
}

func (publisher *OutboxPublisher) Run(ctx context.Context) {
	ticker := time.NewTicker(publisher.interval)
	defer ticker.Stop()

	for {
		if err := publisher.publishBatch(ctx); err != nil && ctx.Err() == nil {
			slog.ErrorContext(ctx, "outbox batch failed", "operation", "outbox_publish", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (publisher *OutboxPublisher) publishBatch(ctx context.Context) error {
	events, err := publisher.store.ClaimOutbox(ctx, publisher.workerID, 25)
	if err != nil {
		return err
	}
	for _, stored := range events {
		var event model.ImportRequestedEvent
		if err := json.Unmarshal(stored.Payload, &event); err != nil {
			_ = publisher.store.ReleaseOutbox(ctx, stored.ID)
			return fmt.Errorf("decode outbox event %s: %w", stored.ID, err)
		}
		eventContext := observability.WithCorrelationID(ctx, event.CorrelationID)
		eventContext = observability.WithTenantID(eventContext, event.TenantID)
		if err := publisher.broker.Publish(eventContext, event); err != nil {
			_ = publisher.store.ReleaseOutbox(ctx, stored.ID)
			return fmt.Errorf("publish outbox event %s: %w", stored.ID, err)
		}
		if err := publisher.store.MarkOutboxPublished(eventContext, stored.ID); err != nil {
			return fmt.Errorf("mark outbox event %s: %w", stored.ID, err)
		}
		slog.InfoContext(eventContext, "outbox event published",
			append(observability.LogAttributes(eventContext),
				"operation", "outbox_publish",
				"event_type", stored.EventType,
				"import_id", event.ImportID,
			)...,
		)
	}
	return nil
}

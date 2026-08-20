package processor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/asafeferreira/devops-observability-lab/internal/model"
	"github.com/asafeferreira/devops-observability-lab/internal/observability"
)

const integrationTarget = "integration-simulator"

type IntegrationClient struct {
	baseURL string
	client  *http.Client
	breaker *CircuitBreaker
	metrics *observability.Metrics
}

func NewIntegrationClient(baseURL string, timeout time.Duration, metrics *observability.Metrics) *IntegrationClient {
	return &IntegrationClient{
		baseURL: baseURL,
		client: &http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
			Timeout:   timeout,
		},
		breaker: NewCircuitBreaker(integrationTarget, 3, 15*time.Second, metrics),
		metrics: metrics,
	}
}

func (client *IntegrationClient) Process(ctx context.Context, item model.Import) error {
	if err := client.breaker.Allow(); err != nil {
		client.metrics.IntegrationCalls.WithLabelValues(integrationTarget, "process_import", "circuit_open").Inc()
		return err
	}

	var lastError error
	for attempt := 1; attempt <= 3; attempt++ {
		if attempt > 1 {
			client.metrics.Retries.WithLabelValues(integrationTarget, "process_import").Inc()
			backoff := time.Duration(attempt-1) * 200 * time.Millisecond
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				client.breaker.Failure()
				return ctx.Err()
			case <-timer.C:
			}
		}

		started := time.Now()
		err := client.call(ctx, item)
		result := "success"
		if err != nil {
			result = "failure"
		}
		client.metrics.IntegrationCalls.WithLabelValues(integrationTarget, "process_import", result).Inc()
		client.metrics.IntegrationDuration.WithLabelValues(integrationTarget, "process_import").Observe(time.Since(started).Seconds())
		if err == nil {
			client.breaker.Success()
			return nil
		}
		lastError = err
		slog.WarnContext(ctx, "integration call failed",
			append(observability.LogAttributes(ctx),
				"operation", "process_import",
				"target", integrationTarget,
				"attempt", attempt,
				"error", err,
			)...,
		)
	}

	client.breaker.Failure()
	return fmt.Errorf("integration failed after retries: %w", lastError)
}

func (client *IntegrationClient) call(ctx context.Context, item model.Import) error {
	body, err := json.Marshal(map[string]any{
		"importId":    item.ID,
		"source":      item.Source,
		"recordCount": item.RecordCount,
	})
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+"/api/v1/process", bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Correlation-ID", observability.CorrelationID(ctx))
	request.Header.Set("X-Tenant-ID", item.TenantID)

	response, err := client.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64*1024))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("integration returned HTTP %d", response.StatusCode)
	}
	return nil
}

func IsCircuitOpen(err error) bool { return errors.Is(err, ErrCircuitOpen) }

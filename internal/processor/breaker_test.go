package processor

import (
	"errors"
	"testing"
	"time"

	"github.com/asafeferreira/devops-observability-lab/internal/observability"
)

func TestCircuitBreakerOpensAndRecovers(t *testing.T) {
	metrics := observability.NewMetrics("test")
	breaker := NewCircuitBreaker("simulator", 2, time.Second, metrics)
	now := time.Date(2026, time.August, 19, 12, 0, 0, 0, time.UTC)
	breaker.clock = func() time.Time { return now }

	breaker.Failure()
	if err := breaker.Allow(); err != nil {
		t.Fatalf("breaker opened too early: %v", err)
	}
	breaker.Failure()
	if err := breaker.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected open breaker, got %v", err)
	}

	now = now.Add(2 * time.Second)
	if err := breaker.Allow(); err != nil {
		t.Fatalf("expected half-open probe, got %v", err)
	}
	if err := breaker.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected concurrent half-open call to be rejected, got %v", err)
	}
	breaker.Success()
	if err := breaker.Allow(); err != nil {
		t.Fatalf("expected closed breaker, got %v", err)
	}
}

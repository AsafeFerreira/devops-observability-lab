package processor

import (
	"errors"
	"sync"
	"time"

	"github.com/asafeferreira/devops-observability-lab/internal/observability"
)

var ErrCircuitOpen = errors.New("integration circuit breaker is open")

type breakerState int

const (
	breakerClosed breakerState = iota
	breakerHalfOpen
	breakerOpen
)

type CircuitBreaker struct {
	mu               sync.Mutex
	state            breakerState
	failures         int
	failureThreshold int
	openFor          time.Duration
	openedAt         time.Time
	probeInFlight    bool
	metrics          *observability.Metrics
	target           string
	clock            func() time.Time
}

func NewCircuitBreaker(target string, threshold int, openFor time.Duration, metrics *observability.Metrics) *CircuitBreaker {
	breaker := &CircuitBreaker{
		state:            breakerClosed,
		failureThreshold: threshold,
		openFor:          openFor,
		metrics:          metrics,
		target:           target,
		clock:            time.Now,
	}
	breaker.recordState()
	return breaker
}

func (breaker *CircuitBreaker) Allow() error {
	breaker.mu.Lock()
	defer breaker.mu.Unlock()

	if breaker.state == breakerOpen && breaker.clock().Sub(breaker.openedAt) >= breaker.openFor {
		breaker.state = breakerHalfOpen
		breaker.probeInFlight = false
		breaker.recordState()
	}
	if breaker.state == breakerOpen || (breaker.state == breakerHalfOpen && breaker.probeInFlight) {
		return ErrCircuitOpen
	}
	if breaker.state == breakerHalfOpen {
		breaker.probeInFlight = true
	}
	return nil
}

func (breaker *CircuitBreaker) Success() {
	breaker.mu.Lock()
	defer breaker.mu.Unlock()

	breaker.state = breakerClosed
	breaker.failures = 0
	breaker.probeInFlight = false
	breaker.recordState()
}

func (breaker *CircuitBreaker) Failure() {
	breaker.mu.Lock()
	defer breaker.mu.Unlock()

	breaker.probeInFlight = false
	breaker.failures++
	if breaker.state == breakerHalfOpen || breaker.failures >= breaker.failureThreshold {
		breaker.state = breakerOpen
		breaker.openedAt = breaker.clock()
	}
	breaker.recordState()
}

func (breaker *CircuitBreaker) recordState() {
	breaker.metrics.CircuitBreakerState.WithLabelValues(breaker.target).Set(float64(breaker.state))
}

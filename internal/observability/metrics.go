package observability

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	Registry            *prometheus.Registry
	HTTPRequests        *prometheus.CounterVec
	HTTPDuration        *prometheus.HistogramVec
	BusinessOperations  *prometheus.CounterVec
	BusinessDuration    *prometheus.HistogramVec
	IntegrationCalls    *prometheus.CounterVec
	IntegrationDuration *prometheus.HistogramVec
	Retries             *prometheus.CounterVec
	CircuitBreakerState *prometheus.GaugeVec
	QueueMessages       *prometheus.CounterVec
	DBQueries           *prometheus.CounterVec
	DBQueryDuration     *prometheus.HistogramVec
	ImportsCurrent      *prometheus.GaugeVec
	WorkerLastSuccess   prometheus.Gauge
	WorkerHeartbeat     prometheus.Gauge
}

func NewMetrics(service string) *Metrics {
	registry := prometheus.NewRegistry()
	registerer := prometheus.WrapRegistererWith(prometheus.Labels{"service": service}, registry)
	registerer.MustRegister(
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)

	metrics := &Metrics{
		Registry: registry,
		HTTPRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "lab_http_requests_total",
			Help: "Total HTTP requests by normalized route, method and status.",
		}, []string{"route", "method", "status"}),
		HTTPDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "lab_http_request_duration_seconds",
			Help:    "HTTP request duration by normalized route and method.",
			Buckets: prometheus.DefBuckets,
		}, []string{"route", "method"}),
		BusinessOperations: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "lab_business_operations_total",
			Help: "Business operations by operation and result.",
		}, []string{"operation", "result"}),
		BusinessDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "lab_business_operation_duration_seconds",
			Help:    "Business operation duration.",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
		}, []string{"operation", "result"}),
		IntegrationCalls: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "lab_integration_calls_total",
			Help: "Calls to external integrations by target, operation and result.",
		}, []string{"target", "operation", "result"}),
		IntegrationDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "lab_integration_duration_seconds",
			Help:    "External integration call duration.",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
		}, []string{"target", "operation"}),
		Retries: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "lab_retries_total",
			Help: "Retry attempts by target and operation.",
		}, []string{"target", "operation"}),
		CircuitBreakerState: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "lab_circuit_breaker_state",
			Help: "Circuit breaker state: 0 closed, 1 half-open, 2 open.",
		}, []string{"target"}),
		QueueMessages: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "lab_queue_messages_total",
			Help: "Queue messages by action and result.",
		}, []string{"action", "result"}),
		DBQueries: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "lab_db_queries_total",
			Help: "Database operations by logical operation and result.",
		}, []string{"operation", "result"}),
		DBQueryDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "lab_db_query_duration_seconds",
			Help:    "Database operation duration without raw SQL labels.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		}, []string{"operation"}),
		ImportsCurrent: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "lab_imports_current",
			Help: "Current imports by status.",
		}, []string{"status"}),
		WorkerLastSuccess: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "lab_worker_last_success_timestamp_seconds",
			Help: "Unix timestamp of the last successful import processing.",
		}),
		WorkerHeartbeat: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "lab_worker_heartbeat_timestamp_seconds",
			Help: "Unix timestamp of the worker heartbeat.",
		}),
	}

	registerer.MustRegister(
		metrics.HTTPRequests,
		metrics.HTTPDuration,
		metrics.BusinessOperations,
		metrics.BusinessDuration,
		metrics.IntegrationCalls,
		metrics.IntegrationDuration,
		metrics.Retries,
		metrics.CircuitBreakerState,
		metrics.QueueMessages,
		metrics.DBQueries,
		metrics.DBQueryDuration,
		metrics.ImportsCurrent,
		metrics.WorkerLastSuccess,
		metrics.WorkerHeartbeat,
	)
	return metrics
}

func (metrics *Metrics) RegisterPool(pool *pgxpool.Pool) {
	metrics.Registry.MustRegister(newPoolCollector(pool))
}

type poolCollector struct {
	pool        *pgxpool.Pool
	connections *prometheus.Desc
}

func newPoolCollector(pool *pgxpool.Pool) *poolCollector {
	return &poolCollector{
		pool: pool,
		connections: prometheus.NewDesc(
			"lab_db_pool_connections",
			"PostgreSQL pool connections by state.",
			[]string{"state"}, nil,
		),
	}
}

func (collector *poolCollector) Describe(channel chan<- *prometheus.Desc) {
	channel <- collector.connections
}

func (collector *poolCollector) Collect(channel chan<- prometheus.Metric) {
	stats := collector.pool.Stat()
	values := map[string]float64{
		"acquired": float64(stats.AcquiredConns()),
		"idle":     float64(stats.IdleConns()),
		"total":    float64(stats.TotalConns()),
		"max":      float64(stats.MaxConns()),
	}
	for state, value := range values {
		channel <- prometheus.MustNewConstMetric(collector.connections, prometheus.GaugeValue, value, state)
	}
}

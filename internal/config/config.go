package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServiceName       string
	Environment       string
	Port              string
	DatabaseURL       string
	RabbitMQURL       string
	IntegrationURL    string
	OTLPEndpoint      string
	LogLevel          string
	LabMode           bool
	OutboxInterval    time.Duration
	ProcessingTimeout time.Duration
}

func Load(serviceName, defaultPort string) Config {
	return Config{
		ServiceName:       serviceName,
		Environment:       value("DEPLOYMENT_ENVIRONMENT", "local"),
		Port:              value("PORT", defaultPort),
		DatabaseURL:       value("DATABASE_URL", "postgres://observability:observability_dev@localhost:5432/observability_lab?sslmode=disable"),
		RabbitMQURL:       value("RABBITMQ_URL", "amqp://observability:observability_dev@localhost:5672/"),
		IntegrationURL:    value("INTEGRATION_URL", "http://localhost:8082"),
		OTLPEndpoint:      value("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4318"),
		LogLevel:          value("LOG_LEVEL", "info"),
		LabMode:           boolean("LAB_MODE", false),
		OutboxInterval:    duration("OUTBOX_INTERVAL", time.Second),
		ProcessingTimeout: duration("PROCESSING_TIMEOUT", 5*time.Second),
	}
}

func value(key, fallback string) string {
	if current := os.Getenv(key); current != "" {
		return current
	}
	return fallback
}

func boolean(key string, fallback bool) bool {
	current := os.Getenv(key)
	if current == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(current)
	if err != nil {
		return fallback
	}
	return parsed
}

func duration(key string, fallback time.Duration) time.Duration {
	current := os.Getenv(key)
	if current == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(current)
	if err != nil {
		return fallback
	}
	return parsed
}

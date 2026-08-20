package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/asafeferreira/devops-observability-lab/internal/config"
	"github.com/asafeferreira/devops-observability-lab/internal/database"
	"github.com/asafeferreira/devops-observability-lab/internal/httpapi"
	"github.com/asafeferreira/devops-observability-lab/internal/messaging"
	"github.com/asafeferreira/devops-observability-lab/internal/observability"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	configuration := config.Load("imports-api", "8080")

	shutdownTelemetry, err := observability.Setup(ctx, configuration.ServiceName, configuration.Environment, configuration.OTLPEndpoint, configuration.LogLevel)
	if err != nil {
		slog.Error("could not configure telemetry", "error", err)
		os.Exit(1)
	}
	defer func() { _ = observability.ShutdownWithTimeout(shutdownTelemetry) }()

	metrics := observability.NewMetrics(configuration.ServiceName)
	store, err := database.OpenWithRetry(ctx, configuration.DatabaseURL, metrics)
	if err != nil {
		slog.Error("could not connect to database", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		slog.Error("could not migrate database", "error", err)
		os.Exit(1)
	}

	broker, err := messaging.ConnectWithRetry(ctx, configuration.RabbitMQURL, metrics)
	if err != nil {
		slog.Error("could not connect to RabbitMQ", "error", err)
		os.Exit(1)
	}
	defer func() { _ = broker.Close() }()
	go messaging.NewOutboxPublisher(store, broker, configuration.OutboxInterval).Run(ctx)

	server := &http.Server{
		Addr:              ":" + configuration.Port,
		Handler:           otelhttp.NewHandler(httpapi.New(store, broker, metrics), "imports-api.http"),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go shutdownOnSignal(ctx, server)
	slog.Info("service started", "operation", "startup", "port", configuration.Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("HTTP server failed", "error", err)
		os.Exit(1)
	}
}

func shutdownOnSignal(ctx context.Context, server *http.Server) {
	<-ctx.Done()
	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		slog.Error("HTTP shutdown failed", "error", err)
	}
}

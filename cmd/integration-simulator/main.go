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
	"github.com/asafeferreira/devops-observability-lab/internal/observability"
	"github.com/asafeferreira/devops-observability-lab/internal/simulator"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	configuration := config.Load("integration-simulator", "8082")

	shutdownTelemetry, err := observability.Setup(ctx, configuration.ServiceName, configuration.Environment, configuration.OTLPEndpoint, configuration.LogLevel)
	if err != nil {
		slog.Error("could not configure telemetry", "error", err)
		os.Exit(1)
	}
	defer func() { _ = observability.ShutdownWithTimeout(shutdownTelemetry) }()
	metrics := observability.NewMetrics(configuration.ServiceName)
	server := &http.Server{
		Addr:              ":" + configuration.Port,
		Handler:           otelhttp.NewHandler(simulator.New(configuration.LabMode, metrics), "integration-simulator.http"),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			slog.Error("HTTP shutdown failed", "error", err)
		}
	}()
	slog.Info("service started", "operation", "startup", "port", configuration.Port, "lab_mode", configuration.LabMode)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("HTTP server failed", "error", err)
		os.Exit(1)
	}
}

package observability

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

type Shutdown func(context.Context) error

func Setup(ctx context.Context, serviceName, environment, endpoint, configuredLevel string) (Shutdown, error) {
	level := slog.LevelInfo
	if strings.EqualFold(configuredLevel, "debug") {
		level = slog.LevelDebug
	}

	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		semconv.DeploymentEnvironmentNameKey.String(environment),
		attribute.String("service.version", "dev"),
	))
	if err != nil {
		return nil, err
	}

	endpoint = normalizeEndpoint(endpoint)
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.AlwaysSample())),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logExporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(endpoint),
		otlploghttp.WithInsecure(),
	)
	if err != nil {
		_ = tracerProvider.Shutdown(ctx)
		return nil, err
	}
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
	)
	global.SetLoggerProvider(loggerProvider)

	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}).WithAttrs([]slog.Attr{
		slog.String("service", serviceName),
		slog.String("environment", environment),
	})
	otelHandler := otelslog.NewHandler(serviceName,
		otelslog.WithLoggerProvider(loggerProvider),
	)
	slog.SetDefault(slog.New(multiHandler{handlers: []slog.Handler{jsonHandler, otelHandler}}))

	return func(shutdownContext context.Context) error {
		return errors.Join(
			loggerProvider.Shutdown(shutdownContext),
			tracerProvider.Shutdown(shutdownContext),
		)
	}, nil
}

func normalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	return strings.TrimSuffix(endpoint, "/")
}

type multiHandler struct {
	handlers []slog.Handler
}

func (handler multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, child := range handler.handlers {
		if child.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (handler multiHandler) Handle(ctx context.Context, record slog.Record) error {
	var combined error
	for _, child := range handler.handlers {
		if child.Enabled(ctx, record.Level) {
			combined = errors.Join(combined, child.Handle(ctx, record.Clone()))
		}
	}
	return combined
}

func (handler multiHandler) WithAttrs(attributes []slog.Attr) slog.Handler {
	children := make([]slog.Handler, 0, len(handler.handlers))
	for _, child := range handler.handlers {
		children = append(children, child.WithAttrs(attributes))
	}
	return multiHandler{handlers: children}
}

func (handler multiHandler) WithGroup(name string) slog.Handler {
	children := make([]slog.Handler, 0, len(handler.handlers))
	for _, child := range handler.handlers {
		children = append(children, child.WithGroup(name))
	}
	return multiHandler{handlers: children}
}

func ShutdownWithTimeout(shutdown Shutdown) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return shutdown(ctx)
}

package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

type contextKey string

const (
	correlationKey contextKey = "correlation_id"
	tenantKey      contextKey = "tenant_id"
)

var safeContextValue = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

func WithCorrelationID(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, correlationKey, value)
}

func CorrelationID(ctx context.Context) string {
	value, _ := ctx.Value(correlationKey).(string)
	return value
}

func WithTenantID(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, tenantKey, value)
}

func TenantID(ctx context.Context) string {
	value, _ := ctx.Value(tenantKey).(string)
	return value
}

func SanitizeCorrelationID(value string) string {
	value = strings.TrimSpace(value)
	if safeContextValue.MatchString(value) {
		return value
	}
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "correlation-unavailable"
	}
	return hex.EncodeToString(buffer)
}

func LogAttributes(ctx context.Context) []any {
	attributes := make([]any, 0, 8)
	if value := CorrelationID(ctx); value != "" {
		attributes = append(attributes, "correlation_id", value)
	}
	if value := TenantID(ctx); value != "" {
		attributes = append(attributes, "tenant_id", value)
	}
	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.IsValid() {
		attributes = append(attributes,
			"trace_id", spanContext.TraceID().String(),
			"span_id", spanContext.SpanID().String(),
		)
	}
	return attributes
}

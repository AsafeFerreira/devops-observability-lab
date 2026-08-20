package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/asafeferreira/devops-observability-lab/internal/model"
	"github.com/asafeferreira/devops-observability-lab/internal/observability"
)

const (
	ExchangeName   = "lab.imports"
	QueueName      = "lab.imports.process"
	RoutingKey     = "import.requested"
	DeadExchange   = "lab.imports.dlx"
	DeadQueue      = "lab.imports.dead_letter"
	DeadRoutingKey = "import.dead_letter"
)

type Broker struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	confirms   <-chan amqp.Confirmation
	metrics    *observability.Metrics
	tracer     trace.Tracer
}

func ConnectWithRetry(ctx context.Context, url string, metrics *observability.Metrics) (*Broker, error) {
	var lastError error
	for attempt := 1; attempt <= 15; attempt++ {
		connection, err := amqp.DialConfig(url, amqp.Config{
			Heartbeat: 10 * time.Second,
			Dial:      amqp.DefaultDial(5 * time.Second),
		})
		if err == nil {
			channel, channelError := connection.Channel()
			if channelError == nil {
				broker := &Broker{
					connection: connection,
					channel:    channel,
					metrics:    metrics,
					tracer:     otel.Tracer("lab/messaging"),
				}
				if setupError := broker.setup(); setupError == nil {
					return broker, nil
				} else {
					lastError = setupError
				}
				_ = channel.Close()
			}
			_ = connection.Close()
			if channelError != nil {
				lastError = channelError
			}
		} else {
			lastError = err
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(attempt) * 200 * time.Millisecond):
		}
	}
	return nil, fmt.Errorf("connect to RabbitMQ after retries: %w", lastError)
}

func (broker *Broker) setup() error {
	if err := broker.channel.ExchangeDeclare(ExchangeName, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare imports exchange: %w", err)
	}
	if err := broker.channel.ExchangeDeclare(DeadExchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dead-letter exchange: %w", err)
	}
	queue, err := broker.channel.QueueDeclare(QueueName, true, false, false, false, amqp.Table{
		"x-dead-letter-exchange":    DeadExchange,
		"x-dead-letter-routing-key": DeadRoutingKey,
	})
	if err != nil {
		return fmt.Errorf("declare process queue: %w", err)
	}
	if err := broker.channel.QueueBind(queue.Name, RoutingKey, ExchangeName, false, nil); err != nil {
		return fmt.Errorf("bind process queue: %w", err)
	}
	deadQueue, err := broker.channel.QueueDeclare(DeadQueue, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("declare dead-letter queue: %w", err)
	}
	if err := broker.channel.QueueBind(deadQueue.Name, DeadRoutingKey, DeadExchange, false, nil); err != nil {
		return fmt.Errorf("bind dead-letter queue: %w", err)
	}
	if err := broker.channel.Qos(10, 0, false); err != nil {
		return fmt.Errorf("configure consumer qos: %w", err)
	}
	if err := broker.channel.Confirm(false); err != nil {
		return fmt.Errorf("enable publisher confirms: %w", err)
	}
	broker.confirms = broker.channel.NotifyPublish(make(chan amqp.Confirmation, 1))
	return nil
}

func (broker *Broker) Close() error {
	return errors.Join(broker.channel.Close(), broker.connection.Close())
}

func (broker *Broker) Healthy() bool {
	return broker != nil && !broker.connection.IsClosed()
}

func (broker *Broker) Publish(ctx context.Context, event model.ImportRequestedEvent) (err error) {
	parentContext := otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(event.TraceContext))
	ctx, span := broker.tracer.Start(parentContext, "rabbitmq publish import.requested",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "rabbitmq"),
			attribute.String("messaging.destination.name", QueueName),
			attribute.String("messaging.operation.name", "publish"),
		),
	)
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "publish failed")
		}
		span.End()
	}()

	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	headers := amqp.Table{
		"correlation_id": event.CorrelationID,
		"tenant_id":      event.TenantID,
	}
	otel.GetTextMapPropagator().Inject(ctx, tableCarrier{table: headers})

	if err := broker.channel.PublishWithContext(ctx, ExchangeName, RoutingKey, false, false, amqp.Publishing{
		Headers:       headers,
		ContentType:   "application/json",
		DeliveryMode:  amqp.Persistent,
		MessageId:     event.ImportID,
		CorrelationId: event.CorrelationID,
		Type:          "import.requested",
		Timestamp:     time.Now().UTC(),
		Body:          body,
	}); err != nil {
		broker.metrics.QueueMessages.WithLabelValues("publish", "failure").Inc()
		return err
	}

	select {
	case confirmation, ok := <-broker.confirms:
		if !ok || !confirmation.Ack {
			broker.metrics.QueueMessages.WithLabelValues("publish", "failure").Inc()
			return errors.New("RabbitMQ did not confirm the message")
		}
		broker.metrics.QueueMessages.WithLabelValues("publish", "success").Inc()
		return nil
	case <-ctx.Done():
		broker.metrics.QueueMessages.WithLabelValues("publish", "failure").Inc()
		return ctx.Err()
	}
}

func (broker *Broker) Consume(ctx context.Context) (<-chan amqp.Delivery, error) {
	return broker.channel.ConsumeWithContext(ctx, QueueName, "", false, false, false, false, nil)
}

func ExtractDeliveryContext(ctx context.Context, delivery amqp.Delivery) context.Context {
	ctx = otel.GetTextMapPropagator().Extract(ctx, tableCarrier{table: delivery.Headers})
	if delivery.CorrelationId != "" {
		ctx = observability.WithCorrelationID(ctx, observability.SanitizeCorrelationID(delivery.CorrelationId))
	}
	if tenant, ok := delivery.Headers["tenant_id"].(string); ok {
		ctx = observability.WithTenantID(ctx, tenant)
	}
	return ctx
}

type tableCarrier struct {
	table amqp.Table
}

func (carrier tableCarrier) Get(key string) string {
	value, _ := carrier.table[key].(string)
	return value
}

func (carrier tableCarrier) Set(key, value string) { carrier.table[key] = value }

func (carrier tableCarrier) Keys() []string {
	keys := make([]string, 0, len(carrier.table))
	for key := range carrier.table {
		keys = append(keys, key)
	}
	return keys
}

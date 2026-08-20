# ADR 0003: telemetria, correlação e cardinalidade

- Status: aceito
- Data: 2026-08-19

## Contexto

O painel precisa separar logs por cliente, mas labels Prometheus ou Loki sem limite podem explodir cardinalidade. Também é necessário navegar entre requisição, banco, fila e integração.

## Decisão

- Propagar W3C Trace Context em HTTP e headers AMQP.
- Manter `tenant_id`, `correlation_id`, `trace_id` e `span_id` como atributos estruturados de log.
- Promover apenas resource attributes estáveis, como `service.name`, a labels Loki.
- Não usar tenant, UUID, correlation ID, URL arbitrária ou SQL como labels Prometheus.
- Adicionar trace ID como exemplar de histogramas HTTP.
- Exportar logs via bridge `slog` → OTel e simultaneamente JSON stdout.
- Amostrar 100% somente no laboratório; produção precisa de decisão de volume/custo e tail/head sampling.

## Maturidade

No OpenTelemetry Go, traces e métricas têm maturidade estável, enquanto o sinal de logs e integrações relacionadas evoluem em ritmo diferente. O stdout JSON mantém diagnóstico independente e reduz acoplamento ao exporter. Upgrades devem validar API, mapeamento OTLP→Loki e dashboards.

## Consequências

- filtro por cliente funciona nos logs sem multiplicar séries métricas;
- correlação é direta por trace e exemplar;
- consulta por tenant depende de structured metadata no Loki;
- logs duplicam saída local e OTLP intencionalmente;
- isolamento de tenant não é autenticação e precisa ser implementado separadamente em produção.

Referências: [OpenTelemetry Go](https://opentelemetry.io/docs/languages/go/), [Loki OTLP](https://grafana.com/docs/loki/latest/send-data/otel/), [Prometheus exemplars](https://prometheus.io/docs/specs/om/open_metrics_spec/#exemplars).

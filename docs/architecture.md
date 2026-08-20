# Arquitetura

## Objetivo

O laboratório representa uma plataforma web multiusuário que recebe solicitações de importação, registra o estado no PostgreSQL e processa cada solicitação de forma assíncrona. O domínio é intencionalmente pequeno: a ênfase está no comportamento operacional observável.

## Componentes

| Componente | Responsabilidade | Porta |
|---|---|---:|
| `imports-api` | valida tenant/idempotência, persiste import e outbox, expõe consulta | 8080 |
| `imports-worker` | consome RabbitMQ, executa integração, atualiza status | 8081 |
| `integration-simulator` | simula dependência normal, lenta, intermitente ou indisponível | 8082 interna / 18082 no host Compose |
| PostgreSQL | estado transacional, outbox, constraints e índices | 5432 |
| RabbitMQ | transporte, confirmação, QoS e DLQ | 5672 |
| OTel Collector | recebe e encaminha logs e traces | 4317/4318 |
| Prometheus | coleta métricas, avalia regras e recebe span metrics | 9090 |
| Loki | armazena logs OTLP e metadata estruturada | 3100 |
| Tempo | armazena traces e gera service graphs/span metrics | 3200 |
| Alertmanager | agrupa, inibe e encaminha alertas | 9093 |
| Grafana | visualização e correlação dos três sinais | 3000 |

## Fluxo de importação

1. O cliente envia `POST /api/v1/imports` com `X-Tenant-ID` e `Idempotency-Key`.
2. A API normaliza e valida os identificadores.
3. Uma única transação cria a importação e um evento no outbox. Repetir a mesma chave para o mesmo cliente devolve o recurso original.
4. O publicador do outbox bloqueia lotes com `FOR UPDATE SKIP LOCKED`, publica uma mensagem persistente e espera publisher confirm.
5. O worker consome com ack manual e QoS limitado.
6. A integração é chamada até três vezes, com backoff. Três operações definitivamente malsucedidas abrem o circuit breaker por 15 segundos.
7. Sucesso termina em `SUCCEEDED` + ack. Falha final termina em `DEAD_LETTER` + nack sem requeue; o RabbitMQ encaminha a mensagem à DLQ.

Estados persistidos:

```text
QUEUED -> PROCESSING -> SUCCEEDED
                    \-> DEAD_LETTER
```

`FAILED` permanece no schema para evolução futura com retry agendado; a implementação atual usa retries imediatos na integração e finaliza diretamente em DLQ.

## Por que transactional outbox

Sem outbox, uma queda entre o commit no banco e a publicação produziria uma importação permanentemente parada. A transação grava estado e intenção de publicação juntas. O publicador aceita entrega ao menos uma vez, e o processamento se apoia em estado persistido/idempotência. A escolha está registrada no [ADR 0002](decisions/0002-transactional-outbox.md).

## Correlação dos sinais

```text
HTTP traceparent
  -> span da API
  -> spans de operações PostgreSQL
  -> contexto gravado no evento/outbox
  -> headers AMQP traceparent/tracestate
  -> span consumer no worker
  -> span HTTP client/server da integração
```

Cada log recebe, quando disponível, `service.name`, `deployment.environment.name`, `tenant_id`, `correlation_id`, `trace_id` e `span_id`. Histogramas HTTP armazenam `trace_id` como exemplar. No Grafana:

- exemplar de métrica abre o trace no Tempo;
- o trace abre os logs do mesmo serviço/janela no Loki;
- o campo estruturado `trace_id` do log abre o trace;
- service graphs vêm das span metrics geradas pelo Tempo.

## Multi-tenancy e cardinalidade

`tenant_id` é metadata estruturada no Loki e filtro de dashboard, mas não é label de métricas. Isso permite separar clientes sem criar uma série Prometheus por cliente, rota e status. IDs de importação e correlação também nunca são labels Prometheus.

O laboratório não implementa isolamento criptográfico entre tenants. `X-Tenant-ID` é um seletor didático, não autenticação. Produção exigiria identidade validada e autorização server-side.

## Confiabilidade

- timeout explícito em servidor e cliente HTTP;
- retries limitados e circuit breaker;
- mensagens persistentes, publisher confirms, ack manual e DLQ;
- migrações protegidas por PostgreSQL advisory lock;
- readiness inclui banco e broker; liveness verifica o processo;
- limites de payload, rotas normalizadas nas métricas e erros sem detalhes internos;
- graceful shutdown com prazo de dez segundos.

## Formas de execução

Docker Compose é o caminho rápido e persistente por named volumes. O overlay `kind` demonstra Deployments/StatefulSets, ServiceMonitor, PrometheusRule, probes, requests/limits, non-root e Kustomize. O ambiente kind usa `emptyDir` deliberadamente e não representa persistência de produção.

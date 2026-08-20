# Observabilidade

## Sinais

| Sinal | Origem | Transporte | Destino |
|---|---|---|---|
| Logs | `slog` + bridge OTel | OTLP/HTTP | Collector → Loki |
| Traces | OTel HTTP, banco e mensageria | W3C + OTLP/HTTP | Collector → Tempo |
| Métricas | client Prometheus + exporters | pull/OpenMetrics | Prometheus |
| Span metrics | Tempo metrics-generator | remote write | Prometheus |

Os serviços também escrevem JSON em stdout para diagnóstico quando o pipeline de telemetria não está disponível.

## Métricas próprias

- `lab_http_requests_total` e `lab_http_request_duration_seconds`
- `lab_business_operations_total` e `lab_business_operation_duration_seconds`
- `lab_integration_calls_total` e `lab_integration_duration_seconds`
- `lab_retries_total` e `lab_circuit_breaker_state`
- `lab_queue_messages_total`
- `lab_db_queries_total`, `lab_db_query_duration_seconds` e `lab_db_pool_connections`
- `lab_imports_current`
- `lab_worker_heartbeat_timestamp_seconds` e `lab_worker_last_success_timestamp_seconds`

Labels são listas fechadas de serviço, operação, resultado, rota normalizada, status e estado. Não há tenant, UUID, correlation ID, URL arbitrária nem SQL em labels.

## Dashboards

| UID | Pergunta respondida |
|---|---|
| `lab-overview` | a plataforma está saudável e processando? |
| `lab-tenant-logs` | o que ocorreu com um cliente/componente/correlação? |
| `lab-processing` | há backlog, retries, breaker aberto ou DLQ? |
| `lab-postgresql` | banco e pool estão disponíveis e dentro dos limites? |
| `lab-kubernetes` | pods/nós estão sob pressão ou reiniciando? |

Todos os dashboards ficam em `deploy/grafana/dashboards` e são somente leitura no Grafana provisionado. Uma mudança deve passar por pull request.

## Alertas e teste seguro

| Alerta | Condição resumida | Simulação |
|---|---|---|
| `LabServiceDown` | scrape de serviço falha por 1 min | `simulate-failure.sh stop-worker` |
| `LabHighHTTPErrorRate` | >5% de 5xx com tráfego | usar integração forçada e/ou parar dependência |
| `LabHighHTTPP95Latency` | p95 >750 ms | `simulate-failure.sh latency` repetidamente ou k6 |
| `LabApplicationMemoryHigh` | RSS >256 MiB | observar; não forçar exaustão no host |
| `LabIntegrationFailures` | >3 falhas em 5 min | `simulate-failure.sh integration` várias vezes |
| `LabCircuitBreakerOpen` | estado 2 | três importações `force-error` |
| `LabImportDeadLettered` | import em `DEAD_LETTER` | uma importação `force-error` |
| `LabRabbitQueueBacklog` | >20 prontas por 3 min | parar worker, executar k6, recuperar worker |
| `LabRabbitDeadLetterQueueNotEmpty` | DLQ >0 | uma importação `force-error` |
| `LabWorkerHeartbeatMissing` | heartbeat >60 s | parar worker |
| `LabPostgreSQLDown` | exporter retorna `pg_up=0` | parar/reiniciar PostgreSQL |
| `LabDatabasePoolNearLimit` | pool >80% por 5 min | investigar sob carga, não aumentar artificialmente |

Em Compose, Alertmanager envia estados firing e resolved para `alert-recorder`, que acrescenta registros em `/data/alerts.ndjson`. Leia os últimos eventos com:

```bash
./scripts/simulate-failure.sh evidence
```

## Investigação pelos três sinais

1. Abra um alerta e seu runbook.
2. No overview, identifique o serviço, janela e alteração de taxa/latência.
3. Abra um exemplar do histograma no Tempo.
4. No trace, localize o span de banco, fila ou integração mais lento/com erro.
5. Use trace-to-logs ou o `trace_id` estruturado no dashboard de logs.
6. Aplique o filtro do cliente apenas nos logs; métricas permanecem agregadas.
7. Registre hipótese, ação, horário de recuperação e evidência.

## Retenção local

Prometheus, Loki e Tempo usam sete dias. Esse valor reduz consumo no laptop e não é recomendação de produção. Produção precisa de requisitos legais, custo, backup e armazenamento de objetos.

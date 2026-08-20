# Mapa de evidências para currículo e entrevista

Este documento ajuda a apresentar o projeto com precisão, sem transformar laboratório em falsa experiência de produção.

## Frase curta para currículo

> Desenvolvi um laboratório de observabilidade para uma plataforma Go multi-tenant e assíncrona, com Docker/Kubernetes, PostgreSQL, RabbitMQ, OpenTelemetry, Prometheus, Grafana, Loki e Tempo; automatizei CI, alertas, testes de carga e verificação de backup/restore, com runbooks versionados.

## Pontos demonstráveis

| Competência | Arquivo/comando | O que explicar |
|---|---|---|
| Linux e automação | `scripts/*.sh` | `set -eu`, timeouts, nomes seguros, cleanup com trap |
| Git e revisão | `.github/workflows/ci.yml` | checks obrigatórios antes de merge e artifacts |
| Containers | `Dockerfile`, `compose.yaml` | multi-stage, non-root, health checks, dependências por readiness |
| Kubernetes | `deploy/kubernetes` | Kustomize, probes, resources, ServiceMonitor e PrometheusRule |
| SQL/PostgreSQL | `migrations/001_init.sql` | constraints, índices, transação, advisory lock e outbox |
| Mensageria | `internal/messaging` | confirms, persistência, QoS, ack/nack e DLQ |
| Go | `cmd/`, `internal/` | interfaces, contexto, graceful shutdown, testes e race detector |
| Python | `alert_recorder.py` | webhook concorrente, NDJSON e limite de payload |
| Prometheus | `metrics.go`, `alerts.yaml` | RED/business metrics, histograms, exemplars e cardinalidade |
| Grafana/Loki/Tempo | provisioning e dashboards | correlação métrica → trace → log e filtro por cliente |
| OpenTelemetry | `logging.go`, mensageria e HTTP | propagação W3C entre HTTP, outbox e AMQP |
| Teste de carga | `tests/load/imports.js` | arrival rate, p95 e taxa de falha como thresholds |
| Backup | `verify-backup.sh` | restaurar é a prova do backup; comparação automatizada |
| Documentação | `docs/runbooks`, ADRs e estudos | procedimentos objetivos, decisões e trade-offs |

## Roteiro de demonstração de 8 minutos

1. Mostrar arquitetura e o Compose.
2. Criar uma importação normal; abrir o overview e o trace completo.
3. Filtrar logs por `client-a` e abrir o mesmo trace pelo `trace_id`.
4. Criar `force-error`; mostrar retries, estado `DEAD_LETTER`, DLQ e alerta.
5. Recuperar a causa, aguardar `resolved` e mostrar o NDJSON do alert recorder.
6. Executar `make backup` e abrir o relatório de comparação.
7. Mostrar o workflow periódico e um runbook.
8. Encerrar explicando limitações do laboratório e o que mudaria em produção.

## Perguntas prováveis

**Por que não colocar tenant nas métricas?** Porque um label por cliente multiplica séries e pode causar cardinalidade sem limite. O tenant é apropriado para logs estruturados; métricas agregam comportamento do serviço.

**Por que outbox?** Para não perder a intenção de publicar quando o processo cai depois do commit e antes do RabbitMQ.

**Por que confirmar publicação e usar ack manual?** A confirmação cobre broker recebendo a mensagem; ack manual só a remove depois do efeito esperado no worker.

**Backup bem-sucedido significa arquivo criado?** Não. O script restaura em banco isolado e compara contagens e migrações.

**Isso é produção?** Não. É um laboratório reproduzível com controles inspirados em produção. HA, armazenamento externo, autenticação real, SLOs e gestão externa de segredos ainda seriam necessários.

## Evidência que ainda depende do GitHub

Após publicar, execute os workflows e adicione à candidatura o link de uma execução verde e de seus artifacts. Não afirme que o CI passou no GitHub antes dessa execução existir.

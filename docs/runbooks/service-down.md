# Serviço indisponível

Alertas: `LabServiceDown`, `LabWorkerHeartbeatMissing`, `LabKubernetesDeploymentUnavailable`, `LabKubernetesContainerRestarting`. Severidade: crítica quando há indisponibilidade confirmada.

## Confirmar impacto

1. Confirme o alerta e o horário inicial no Prometheus/Alertmanager.
2. Execute `docker compose ps` ou `kubectl get pods -n observability-lab -o wide`.
3. Teste `/health/live` e `/health/ready`. Liveness falhando indica processo; readiness isolada indica dependência.
4. Consulte logs do componente na janela do alerta e preserve `trace_id`/`correlation_id` relevantes.

## Diagnosticar

- `docker compose logs --since 10m <serviço>` ou `kubectl logs -n observability-lab deployment/<serviço> --since=10m`.
- Verifique eventos Kubernetes com `kubectl get events -n observability-lab --sort-by=.lastTimestamp`.
- Diferencie crash, probe, imagem, configuração, PostgreSQL, RabbitMQ e pressão de recurso.
- Não reinicie repetidamente sem entender se isso amplia perda, duplicação ou backlog.

## Conter e recuperar

No laboratório, recupere um serviço parado com `docker compose start <serviço>` ou reaplique o overlay. Se a causa for dependência, siga o runbook correspondente primeiro. Em produção, reinício/rollback exige autorização e janela acordada.

## Validar

Readiness deve voltar a 200, o target deve ficar `up == 1`, heartbeat deve atualizar, backlog deve diminuir e o alerta deve resolver. Execute `make smoke` e uma importação normal.

## Escalar

Peça apoio se houver restart loop, possível corrupção, alteração de schema/configuração, falha recorrente após recuperação ou necessidade de tocar dados de terceiros.

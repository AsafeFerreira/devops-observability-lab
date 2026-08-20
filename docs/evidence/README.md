# Evidências geradas

Esta pasta documenta evidências; arquivos de execução reais ficam em `artifacts/` e não são versionados para evitar dumps, logs ou dados acidentais no Git.

## Checklist de captura

1. `make up && make smoke` — salve o resultado do terminal.
2. `make integration` — registre IDs de sucesso/idempotência/DLQ.
3. Grafana overview — capture golden signals depois de gerar tráfego.
4. Dashboard de logs — filtre `client-a` e abra um trace pelo `trace_id`.
5. Tempo — capture o trace contendo API, banco, RabbitMQ, worker e simulador.
6. Execute uma falha controlada e capture o alerta firing.
7. Recupere o componente, espere o alerta resolved e execute `simulate-failure.sh evidence`.
8. `make load` — preserve `artifacts/k6-summary.json`.
9. `make backup` — preserve dump e `artifacts/backup-verification.json` com cuidado.
10. Publique o repositório e guarde links dos artifacts dos workflows.

## Regra de integridade

Não há imagens ou resultados pré-fabricados neste repositório. Evidência deve ser produzida por uma execução identificável e nunca conter credenciais, payloads de terceiros ou dumps públicos.

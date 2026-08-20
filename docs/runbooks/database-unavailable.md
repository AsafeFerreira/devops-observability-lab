# PostgreSQL indisponível ou pool saturado

Alertas: `LabPostgreSQLDown`, `LabDatabasePoolNearLimit`. Severidade: critical para indisponibilidade e warning para pool.

## Confirmar

1. Compare `pg_up`, readiness da API/worker e `lab_db_pool_connections`.
2. Execute `pg_isready` no container/pod; não use credencial em linha de comando pública.
3. Verifique conexões por estado, transações, rollbacks, latência e logs do PostgreSQL.
4. Identifique deploy, carga, query ou mudança de rede/configuração no início do evento.

## Ação segura

- Indisponível no laboratório: `docker compose start postgres`.
- Pool alto: procure operação lenta/vazamento antes de aumentar `MaxConns`.
- Não finalize sessões, execute DDL, restaure backup ou faça failover sem orientação do responsável.
- Preserve consistência; não contorne constraints ou outbox.

## Validar

`pg_up == 1`, readiness 200, pool abaixo de 80%, latência normal, outbox publica pendências e uma importação termina com sucesso.

## Escalar

Escale em suspeita de corrupção, perda, disco cheio, locks prolongados, migração falha ou necessidade de restore/failover.

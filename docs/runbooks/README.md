# Runbooks

Procedimentos de primeira resposta para o laboratório. O operador deve preservar dados, evitar mudanças irreversíveis e pedir apoio quando o diagnóstico ultrapassar o escopo acordado.

| Cenário | Runbook |
|---|---|
| serviço ou pod indisponível | [service-down.md](service-down.md) |
| taxa elevada de erros | [high-error-rate.md](high-error-rate.md) |
| latência elevada | [high-latency.md](high-latency.md) |
| pressão de CPU/memória/disco | [resource-pressure.md](resource-pressure.md) |
| falha de integração/breaker | [integration-failure.md](integration-failure.md) |
| backlog do RabbitMQ | [queue-backlog.md](queue-backlog.md) |
| mensagem na DLQ | [dead-letter.md](dead-letter.md) |
| PostgreSQL indisponível/pool saturado | [database-unavailable.md](database-unavailable.md) |
| backup ou restore inválido | [backup-verification-failed.md](backup-verification-failed.md) |

Todo incidente deve registrar: horário, alerta, impacto, tenant(s) afetado(s) quando conhecido, trace/correlation IDs, hipótese, ações, responsável consultado, recuperação e prevenção proposta. Nunca copie senha, token ou dados de cliente para o registro.

# Backlog na fila

Alerta: `LabRabbitQueueBacklog`. Severidade: warning.

## Confirmar

1. Abra o dashboard de processamento e compare mensagens ready, unacked e taxa de consumo.
2. Confirme heartbeat/readiness do worker e saúde da integração/PostgreSQL.
3. Verifique se a entrada aumentou ou o consumo diminuiu.
4. Estime tempo de drenagem: mensagens pendentes ÷ taxa líquida de consumo.

## Ação segura

- Recupere worker/dependência antes de aumentar consumidores.
- Pare carga artificial (`k6`) quando ela estiver causando o cenário.
- Não purgue fila e não mova mensagens manualmente sem autorização e cópia de evidência.
- Escalar réplicas pode ser válido apenas se banco, broker e integração suportarem a nova concorrência.

## Validar

Ready converge para zero, unacked não fica preso, `SUCCEEDED` cresce, latência e erros não pioram. Execute uma importação normal após a drenagem.

## Escalar

Escale quando o backlog continuar crescendo, houver mensagens antigas/SLA em risco, consumo estiver travado ou qualquer ação puder duplicar/perder dados.

# Mensagem na dead-letter queue

Alertas: `LabImportDeadLettered`, `LabRabbitDeadLetterQueueNotEmpty`, `LabKubernetesDeadLetterQueueNotEmpty`. Severidade: crítica porque exige análise de dados.

## Preservar e identificar

1. Não purge a DLQ.
2. Identifique horário, tamanho da fila, import ID, tenant e correlation ID sem divulgar payload.
3. No PostgreSQL, confira status/attempts/last_error pelo endpoint do tenant.
4. Abra trace e logs para localizar causa original e tentativas.

## Classificar

- dependência temporariamente indisponível;
- payload/contrato permanentemente inválido;
- bug de aplicação;
- autenticação/segredo/configuração;
- dado duplicado ou já aplicado.

## Reprocessamento

Este laboratório não oferece replay automático deliberadamente. Antes de replay em qualquer ambiente, confirme que a causa foi corrigida, que o efeito é idempotente e que a equipe/dono do dado autorizou. Registre IDs e quantidade. Uma ferramenta futura deve copiar, não editar silenciosamente, e manter trilha de auditoria.

## Validar

Nova importação controlada alcança `SUCCEEDED`; não há novo crescimento da DLQ; alerta resolve somente depois do tratamento autorizado da mensagem existente.

## Escalar

Sempre peça apoio antes de apagar/republicar mensagens ou alterar dados de terceiros.

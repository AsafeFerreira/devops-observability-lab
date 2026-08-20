# Falha de integração ou circuit breaker aberto

Alertas: `LabIntegrationFailures`, `LabCircuitBreakerOpen`, `LabKubernetesIntegrationFailures`.

## Confirmar

1. Verifique taxa, status e duração das chamadas no dashboard de processamento.
2. Abra um trace e determine se houve timeout, resposta 5xx ou breaker aberto.
3. Consulte logs por `correlation_id` sem copiar o payload.
4. Confirme se a falha é limitada a uma fonte/cliente ou geral.

## Comportamento esperado

Cada importação tenta a integração até três vezes, com backoff de 200/400 ms. Três operações definitivamente malsucedidas abrem o breaker por 15 s. Uma chamada de teste half-open decide fechar ou reabrir. A falha final envia o evento à DLQ e marca a importação `DEAD_LETTER`.

## Ação

No laboratório, recupere o simulador ou use fonte normal. Em produção, confirme a saúde/contrato da integração com a equipe responsável; não aumente retries durante uma indisponibilidade, pois isso amplifica carga.

## Validar

Breaker em estado 0, chamadas bem-sucedidas, latência normal e alerta resolved. Eventos em DLQ não são automaticamente reprocessados: siga o runbook de DLQ.

## Escalar

Escale imediatamente se houver mudança de contrato, autenticação/segredo inválido, risco de duplicidade ou necessidade de reenviar dados de cliente.

# Taxa elevada de erros HTTP

Alerta: `LabHighHTTPErrorRate`. Severidade: warning; elevar para critical se o fluxo principal estiver indisponível.

## Confirmar

1. No overview, identifique serviço, status e início da elevação.
2. Compare volume total e 5xx; baixa amostragem pode distorcer percentuais.
3. Abra um exemplar ou trace com erro e determine o primeiro span que falhou.
4. Cruze com logs por componente e `trace_id`.

## Hipóteses

- banco ou broker indisponível;
- integração externa falhando/timeout;
- mudança recente de código/configuração;
- limite de recurso ou pool;
- payload específico inválido tratado incorretamente.

## Ação segura

Siga o runbook da dependência identificada. Não exponha stack trace ao cliente e não desabilite alerta para ocultar o sintoma. Se uma versão recente for a causa e houver autorização, prefira rollback conhecido a uma alteração improvisada.

## Validar

A taxa 5xx deve permanecer abaixo de 5% pela janela configurada, p95 deve normalizar e uma importação de teste deve alcançar `SUCCEEDED`. Registre trace antes/depois.

## Escalar

Escale quando o erro afetar vários clientes, não houver causa em 15 minutos, envolver dados ou exigir rollback/mudança de produção.

# API

Base URL local: `http://localhost:8080`.

## Cabeçalhos

| Cabeçalho | Obrigatório | Regra |
|---|---|---|
| `X-Tenant-ID` | nas rotas de importação | 2–32 letras minúsculas, números, `_` ou `-`; normalizado para minúsculas |
| `Idempotency-Key` | no POST | 8–128 caracteres alfanuméricos ou `._:-` |
| `X-Correlation-ID` | não | valor seguro até 64 caracteres; é gerado quando ausente/inválido |
| `traceparent` | não | W3C Trace Context propagado automaticamente |

Toda resposta inclui `X-Correlation-ID`.

## Criar importação

`POST /api/v1/imports`

```json
{
  "source": "erp",
  "recordCount": 100
}
```

Resposta `201 Created`:

```json
{
  "id": "848d9186-f199-49b7-a83f-35b44a8a427d",
  "tenantId": "client-a",
  "source": "erp",
  "recordCount": 100,
  "status": "QUEUED",
  "attempts": 0,
  "createdAt": "2026-08-19T18:00:00Z",
  "updatedAt": "2026-08-19T18:00:00Z"
}
```

Repetir tenant + chave devolve `200 OK`, cabeçalho `Idempotent-Replay: true` e o mesmo ID.

## Consultar importação

`GET /api/v1/imports/{id}` com `X-Tenant-ID`. O tenant faz parte do filtro SQL; um cliente não consulta o recurso de outro por essa rota.

## Listar importações

`GET /api/v1/imports?limit=50` com `X-Tenant-ID`. O limite aceito é 1–100; valores fora da faixa usam 50.

## Saúde e métricas

| Endpoint | Semântica |
|---|---|
| `GET /health/live` | processo HTTP está executando |
| `GET /health/ready` | API alcança PostgreSQL e RabbitMQ |
| `GET /metrics` | métricas OpenMetrics para Prometheus |

## Erros

```json
{
  "code": "VALIDATION_ERROR",
  "message": "request validation failed",
  "fields": {
    "recordCount": "must be between 1 and 10000"
  }
}
```

Detalhes internos, SQL, credenciais e payloads da integração não são retornados.

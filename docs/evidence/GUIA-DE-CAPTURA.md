# Guia de captura de evidências

Passo a passo para gerar as imagens que faltam em `docs/evidence/`.
Siga na ordem: os três primeiros itens dependem do ambiente com tráfego,
e os dois últimos dependem de disparar e resolver um alerta.

## Antes de começar

```sh
make up
```

Gere tráfego contínuo em outro terminal, senão os painéis ficam vazios:

```sh
while true; do
  curl -s -o /dev/null -X POST http://localhost:8080/api/v1/imports \
    -H 'Content-Type: application/json' -H 'X-Tenant-ID: client-a' \
    -H "Idempotency-Key: cap-$(date +%s%N)" \
    --data '{"source":"catalog","recordCount":12}'
  sleep 2
done
```

Aguarde cerca de dois minutos antes da primeira captura.

## Acessos

| Interface | Endereço | Credenciais |
| --- | --- | --- |
| Grafana | http://localhost:3000 | `admin` / `observability_dev` |
| Prometheus | http://localhost:9090 | — |
| Alertmanager | http://localhost:9093 | — |
| RabbitMQ | http://localhost:15672 | `observability` / `observability_dev` |

## 1. Dashboard de visão geral

`01 - Platform Overview` — http://localhost:3000/d/lab-overview

Ajuste o intervalo para **Last 15 minutes** no canto superior direito.
Confira que estes painéis têm dados antes de capturar:

- Imports in progress
- Failed imports
- HTTP request rate
- HTTP p95 latency
- Business throughput and failures

Salvar como `docs/evidence/01-platform-overview.png`.

O painel **Recent distributed traces** fica abaixo da dobra: role até o
fim do dashboard antes de capturar. Ele lista os traces recentes e cada
`traceID` abre a cascata de spans no Tempo.

O tile **Firing alerts** conta alertas ativos, e os stat tiles vermelhos
são o comportamento esperado depois de simular um incidente: eles indicam
que a detecção funcionou, não que o ambiente está quebrado.

## 2. Logs por cliente e componente

`02 - Logs by Client and Component` — http://localhost:3000/d/lab-tenant-logs

Este é o painel que responde diretamente ao item 3.1 do edital
("painéis de visualização de logs, separados por cliente e por
componente"). Use o seletor de cliente no topo e capture com um cliente
filtrado, mostrando as linhas de log.

Salvar como `docs/evidence/02-logs-por-cliente.png`.

## 3. Trace distribuído

Grafana → **Explore** → fonte **Tempo** → aba **TraceQL**.

Cole o id de um trace com erro. O trace documentado em
`incident-2026-08-20-dead-letter.md` é:

```
df54d20068d3a9bb2d31617600ec81d3
```

Se ele já tiver expirado da retenção, gere um novo:

```sh
sh scripts/simulate-failure.sh integration
```

e pegue o `trace_id` no log correspondente pelo painel de logs.

Expanda a cascata de spans. A captura deve mostrar os três serviços
(`imports-api`, `imports-worker`, `integration-simulator`) e as três
tentativas de retry marcadas com erro.

Salvar como `docs/evidence/03-trace-distribuido.png`.

## 4. Alerta disparado

Dispare uma indisponibilidade real:

```sh
sh scripts/simulate-failure.sh stop-worker
```

Aguarde cerca de um minuto e abra http://localhost:9093.
O alerta `LabServiceDown` aparece como firing, com link para o runbook.

Salvar como `docs/evidence/04-alerta-firing.png`.

Capture também o painel **Firing alerts** do dashboard de visão geral.

## 5. Alerta resolvido

Restabeleça o serviço:

```sh
sh scripts/simulate-failure.sh recover-worker
```

Aguarde cerca de um minuto. O alerta sai da lista de ativos no
Alertmanager. Para a evidência textual do ciclo completo:

```sh
sh scripts/simulate-failure.sh evidence
```

Esse comando imprime os eventos `firing` e `resolved` com horários
reais, registrados pelo alert recorder.

Salvar como `docs/evidence/05-alerta-resolvido.png`.

## 6. Teste de carga

```sh
make load
```

Ao final, `artifacts/k6-summary.json` traz os números da execução.
Capture o resumo impresso no terminal.

Salvar como `docs/evidence/06-teste-de-carga.png`.

## 7. Verificação de backup

```sh
make backup
```

O script gera o dump, restaura em banco temporário, compara a contagem
de registros e remove apenas os recursos que ele mesmo criou.

Salvar como `docs/evidence/07-backup-restore.png`.

## 8. CI aprovado

https://github.com/AsafeFerreira/devops-observability-lab/actions

Abra a execução mais recente da branch `main` com os quatro jobs em
verde e capture a tela.

Salvar como `docs/evidence/08-ci-aprovado.png`.

## Regra de integridade

Nenhuma imagem deve ser montada, editada ou reaproveitada de outra
fonte. Toda evidência precisa vir de uma execução real e não pode conter
credenciais ou dados de terceiros.

# Latência elevada

Alerta: `LabHighHTTPP95Latency`. Severidade: warning.

## Confirmar

1. Confirme p50/p95, volume e rota; p95 isolado com pouco tráfego pede cautela.
2. Abra um exemplar lento no Tempo.
3. Compare duração de HTTP, PostgreSQL, espera de fila e integração.
4. Verifique CPU, memória, pool e backlog no mesmo intervalo.

## Diagnóstico orientado por componente

- span PostgreSQL: observe pool, conexões, transações e plano antes de criar índice;
- span integração: verifique timeout, retry e breaker;
- fila: verifique ready/unacked e heartbeat do worker;
- aplicação: verifique CPU, memória, GC e mudanças recentes.

## Conter

No laboratório, remova a fonte `slow`, reduza carga e recupere dependências. Não aumente timeout, replicas, pool ou recursos sem medir a causa e o efeito; isso pode apenas deslocar a saturação.

## Validar e escalar

O p95 deve ficar abaixo de 750 ms por cinco minutos, sem crescimento de erro/backlog. Escale se a latência estiver causando timeout, se vários componentes saturarem ou se a correção exigir schema/capacidade de produção.

# ADR 0002: transactional outbox e entrega ao menos uma vez

- Status: aceito
- Data: 2026-08-19

## Contexto

A criação da importação precisa persistir no PostgreSQL e gerar mensagem RabbitMQ. Não há transação atômica simples entre os dois sistemas. Publicar antes do commit pode gerar mensagem sem estado; publicar depois pode perder mensagem em uma queda.

## Decisão

Gravar importação e evento de outbox na mesma transação PostgreSQL. Um publicador separado reivindica eventos com `FOR UPDATE SKIP LOCKED`, publica mensagem persistente, aguarda publisher confirm e só então marca `published_at`.

O modelo é ao menos uma vez. Tenant + idempotency key impedem criação duplicada na API. Efeitos futuros do worker devem ser idempotentes antes de aumentar concorrência/replay.

## Consequências

- queda entre commit e publicação é recuperável;
- queda depois do publish e antes de `published_at` pode republicar;
- outbox exige monitoração, limpeza/retention e teste;
- DLQ exige procedimento humano e replay controlado.

## Alternativas rejeitadas

- chamada RabbitMQ diretamente no handler: janela de perda/inconsistência;
- transação distribuída/2PC: complexidade desproporcional e pouco suporte;
- usar somente PostgreSQL como fila: viável em alguns sistemas, mas não demonstra a mensageria presente no contexto da vaga.

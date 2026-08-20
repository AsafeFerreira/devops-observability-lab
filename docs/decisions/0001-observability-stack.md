# ADR 0001: stack de observabilidade local

- Status: aceito
- Data: 2026-08-19

## Contexto

O projeto precisa demonstrar logs, métricas e traces correlacionados em laptop e em kind, sem depender de SaaS ou credencial externa.

## Decisão

Usar OpenTelemetry Collector como entrada OTLP, Prometheus para métricas/alertas, Loki para logs, Tempo para traces, Alertmanager para roteamento e Grafana para visualização. Todos são provisionados em arquivos versionados.

Fixar Tempo 2.10.7 no modo single-binary/local. Tempo 3 introduz uma arquitetura redesenhada em torno de Kafka para o caminho distribuído; essa complexidade não melhora o objetivo formativo local. A série 2.10 é a última planejada da linha 2.x e mantém o laboratório menor. Uma migração futura deve seguir release notes e testar schema/configuração, não trocar apenas a tag.

## Consequências

- três sinais e correlações ficam reproduzíveis sem serviço pago;
- há vários componentes e consumo de memória no laptop;
- storage local e uma réplica não oferecem HA;
- Compose e kind têm configurações parecidas, mas ainda duplicadas;
- versões exigem atualização deliberada e teste de compatibilidade.

## Alternativas rejeitadas

- SigNoz all-in-one: boa experiência integrada, mas esconderia parte da configuração explícita pedida para o portfólio.
- Elastic/OpenSearch: maior custo local para o escopo.
- SaaS: reduz reprodutibilidade e exige segredo/conta externa.

Referências: [Grafana Tempo releases](https://github.com/grafana/tempo/releases), [milestone Tempo 2.10](https://github.com/grafana/tempo/milestone/8), [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/).

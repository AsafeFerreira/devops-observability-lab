# Contributing

1. Crie uma branch curta a partir de `main`.
2. Não inclua `.env`, dumps, tokens ou dados de terceiros.
3. Execute `make fmt`, `make lint`, `make test` e `make compose-validate`.
4. Para mudanças de infraestrutura, execute também `make up`, `make smoke` e `make integration`.
5. Atualize runbooks, dashboards ou ADRs quando o comportamento operacional mudar.
6. Abra um pull request descrevendo risco, validação e rollback.

Commits devem ser pequenos e explicar a intenção. Alterações de dependência precisam informar o motivo e consultar release notes e avisos de segurança.

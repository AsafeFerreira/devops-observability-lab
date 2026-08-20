# Rotinas operacionais: carga, backup e integração contínua

Resultados de execuções reais em 2026-08-20. Os números vieram dos
artefatos gerados pelos próprios scripts e da API do GitHub Actions.

## Teste de carga (k6)

```sh
make load
```

Cenário `imports`: 10 usuários virtuais, taxa constante de 10 iterações
por segundo durante 30 segundos, submetendo importações à API.

| Métrica | Resultado | Limite configurado |
| --- | --- | --- |
| Iterações completas | 301 | — |
| Requisições falhas | 0,00% | `rate < 1%` |
| Latência p95 | 9,52 ms | `p(95) < 750 ms` |
| Latência média | 5,01 ms | — |
| Latência máxima | 56,51 ms | — |

Verificações funcionais:

- `import accepted` — 301 aprovações, 0 falhas
- `response has import id` — 301 aprovações, 0 falhas

Ambos os thresholds passaram com folga. O relatório completo fica em
`artifacts/k6-summary.json`, versionado como evidência.

## Verificação de backup e restauração

```sh
make backup
```

Saída da execução:

```
OK   backup restored and verified (2262 imports)
```

Registro em `artifacts/backup-verification.json`:

```json
{
  "verifiedAt": "2026-08-20T01:50:47Z",
  "backup": "artifacts/backups/observability_lab-20260820T015045Z.dump",
  "imports": 2262,
  "migrations": 1,
  "result": "pass"
}
```

O script gera o dump, cria um banco temporário, restaura, compara a
contagem de registros e destrói apenas o banco que ele mesmo criou.
Conferido após a execução: restam somente `postgres` e
`observability_lab`, sem resíduos do teste.

## Integração contínua

Execução: https://github.com/AsafeFerreira/devops-observability-lab/actions/runs/32322340716

| Job | Resultado | Duração |
| --- | --- | --- |
| Go quality and tests | aprovado | 17s |
| Configuration as code | aprovado | 10s |
| Dependency security | aprovado | 15s |
| Full-stack integration | aprovado | 2m45s |

O job de integração sobe a stack completa no runner, aguarda os health
checks, executa smoke test e cenários ponta a ponta, verifica o backup e
encerra os containers mesmo em caso de falha.

## Capturas de tela pendentes

- [ ] Terminal com o resumo do k6
- [ ] Terminal com a saída da verificação de backup
- [ ] Página do GitHub Actions com os quatro jobs aprovados

# Evidências

Registros de execuções reais do laboratório. Cada documento traz os
comandos usados, os valores obtidos por consulta às APIs e as capturas
correspondentes.

## Documentos

| Documento | Conteúdo |
| --- | --- |
| [incident-2026-08-20-dead-letter.md](incident-2026-08-20-dead-letter.md) | Falha de integração: métrica, log, trace com retries e alerta |
| [incident-2026-08-20-service-down.md](incident-2026-08-20-service-down.md) | Indisponibilidade: ciclo completo de firing a resolved |
| [operations-2026-08-20.md](operations-2026-08-20.md) | Teste de carga, verificação de backup e integração contínua |
| [GUIA-DE-CAPTURA.md](GUIA-DE-CAPTURA.md) | Passo a passo para reproduzir e recapturar as evidências |

## Imagens

| Arquivo | Conteúdo |
| --- | --- |
| `01-platform-overview.png` | Dashboard de visão geral com golden signals |
| `02-logs-por-cliente.png` | Logs separados por cliente e componente |
| `03-trace-distribuido.png` | Trace atravessando os três serviços, com retries |
| `04-alerta-firing.png` | Alertmanager com alertas ativos |
| `05-alerta-resolvido.png` | Alertmanager após a recuperação |
| `06-teste-de-carga.png` | Resumo do k6 |
| `07-backup-restore.png` | Backup gerado, restaurado e comparado |
| `08-ci-aprovado.png` | Execução do GitHub Actions aprovada |

Para mover capturas do Desktop para esta pasta com os nomes corretos:

```sh
sh scripts/publish-evidence.sh
```

O script também lista o que ainda falta.

## Regra de integridade

Nenhuma imagem foi montada, editada ou reaproveitada de outra fonte.
Toda evidência vem de uma execução identificável e não contém
credenciais nem dados de terceiros. Os números citados nos documentos
foram obtidos por consulta direta às APIs do Prometheus, Loki, Tempo,
Alertmanager e GitHub durante a execução.

Arquivos grandes de execução, como dumps de banco, permanecem fora do
Git. Os relatórios `artifacts/k6-summary.json` e
`artifacts/backup-verification.json` são versionados por serem pequenos
e não conterem dados sensíveis.

# Estudo curto: verificação estática de dependências

## Escopo

O repositório contém módulos Go, imagens Docker, Actions, manifests e scripts. Uma única ferramenta não cobre todas essas superfícies.

| Controle | Cobertura | Quando executar | Limitação principal |
|---|---|---|---|
| `go vet` | defeitos estáticos em Go | todo push/PR | não é scanner completo de segurança |
| [govulncheck](https://go.dev/doc/security/vuln/) | vulnerabilidade alcançável pelo código Go | PR e rotina semanal | depende da Go Vulnerability Database e análise de chamadas |
| [Trivy](https://trivy.dev/docs/latest/) | CVEs, secrets e misconfiguration em repo/imagem/IaC | PR, build e rotina | achados exigem triagem; base de dados precisa atualização |
| [Dependabot](https://docs.github.com/code-security/dependabot) | propostas de atualização para Go, Docker e Actions | semanal | PR automático não prova compatibilidade/segurança |
| SBOM (Syft/SPDX) | inventário de componentes | release | inventário não detecta sozinho alcance ou exploração |

## Estratégia recomendada

1. `gofmt`, `go vet`, testes/race detector bloqueiam defeitos básicos.
2. Trivy em modo filesystem procura vulnerabilidades altas/críticas, secrets e misconfigurations.
3. Build de release gera SBOM e scan da imagem final, não só do repositório.
4. Dependabot abre PRs pequenos; testes de integração e release notes orientam aceite.
5. `govulncheck ./...` entra no CI quando sua versão for fixada e revisada.
6. Exceção precisa registrar CVE, alcance, compensação, responsável e prazo.

## Decisão para este laboratório

O workflow atual usa Trivy + Dependabot e as ferramentas padrão do Go. A próxima melhoria é fixar `govulncheck` e gerar SBOM das três imagens. Isso evita vender “zero vulnerabilidades”: o resultado correto é uma triagem repetível, com limites conhecidos.

## Cuidados com supply chain

- preferir actions oficiais e fixar SHA em um ambiente mais rigoroso;
- proteger `main`, exigir revisão e limitar `GITHUB_TOKEN` a `contents: read` por padrão;
- não executar PR não confiável com secrets;
- assinar imagens/releases e registrar provenance quando houver publicação;
- atualizar imagem base deliberadamente, validando comportamento e CVEs.

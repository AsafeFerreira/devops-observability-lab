# Estudo curto: gestão de segredos

## Problema

Kubernetes `Secret` facilita distribuição, mas o valor em base64 não é criptografia. Git, manifests renderizados, logs, shell history, artifacts e backups são caminhos comuns de vazamento.

## Alternativas

| Opção | Segredo no Git | Fonte de verdade | Pontos fortes | Custo operacional |
|---|---:|---|---|---|
| Secret manual | não deveria | cluster | simples | rotação/auditoria frágeis, drift |
| [SOPS](https://getsops.io/) + KMS/age | ciphertext | Git | GitOps, revisão e histórico criptografado | gestão de chaves e cuidado com decrypt no CI |
| [External Secrets Operator](https://external-secrets.io/) | referência | cloud/Vault | sincronização/rotação, usa backend existente | operador, permissões e disponibilidade do backend |
| [Vault](https://developer.hashicorp.com/vault/docs) | referência | Vault | políticas, auditoria e credenciais dinâmicas | alta complexidade e operação dedicada |
| Secret store nativo da cloud | referência | AWS/GCP/Azure | IAM e auditoria integrados | acoplamento ao provedor |

## Recomendação

Para uma equipe já hospedada em cloud, usar o secret manager nativo + External Secrets Operator, com Workload Identity em vez de chaves estáticas. Para GitOps offline/pequeno, SOPS com KMS/age é alternativa pragmática. Vault só se justifica quando requisitos de múltiplos ambientes, políticas ou credenciais dinâmicas compensarem sua operação.

## Controles mínimos

- criptografia de etcd em repouso e acesso RBAC mínimo;
- identidade separada por workload/ambiente;
- rotação e revogação testadas;
- logs/redaction e bloqueio de secrets em CI;
- nenhum segredo em argumento de processo, URL pública, dashboard ou artifact;
- backup criptografado e acesso auditado;
- procedimento de incidente para revogar, rotacionar e investigar alcance.

## Aplicação ao laboratório

`.env.example` e o overlay `local` têm valores públicos, explicitamente didáticos. Eles não devem ser promovidos. Uma versão cloud removeria `secretGenerator.literals`, instalaria External Secrets e referenciaria nomes estáveis; o aplicativo continuaria lendo variáveis, reduzindo alteração de código.

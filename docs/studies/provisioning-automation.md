# Estudo curto: automação de provisionamento

## Pergunta

Qual abordagem seria mais adequada para evoluir este laboratório local para infraestrutura reproduzível em nuvem?

## Comparação

| Ferramenta | Modelo | Pontos fortes | Custos/riscos | Melhor encaixe |
|---|---|---|---|---|
| [OpenTofu](https://opentofu.org/docs/) | declarativo, estado | ecossistema compatível com HCL/providers, open source sob Linux Foundation | estado precisa backend, lock, proteção e revisão | provisionar cluster, rede, banco e serviços gerenciados |
| [Terraform](https://developer.hashicorp.com/terraform/docs) | declarativo, estado | ecossistema maduro, módulos e amplo conhecimento de mercado | licença/governança e estado exigem decisão explícita | organizações já padronizadas em Terraform/Cloud |
| [Pulumi](https://www.pulumi.com/docs/iac/) | declarativo via linguagens | tipos, abstrações e testes usando Go/Python/TypeScript | abstrações podem esconder recursos; também usa estado | equipe forte em engenharia de software |
| [Ansible](https://docs.ansible.com/) | tarefas desejavelmente idempotentes, sem agente | configuração de SO, procedimentos e inventários legíveis | menos natural para ciclo de vida de recursos cloud complexos | configurar hosts e automatizar rotinas pós-provisionamento |
| Helm/Kustomize | manifests Kubernetes | empacotamento/configuração dentro do cluster | não provisiona rede/cluster/cloud por si só | workloads e observabilidade após cluster existir |

## Critérios

1. revisão de plano antes da mudança;
2. estado remoto com lock, criptografia e backup;
3. separação de ambientes e identidades de menor privilégio;
4. módulos pequenos, versões fixadas e política em CI;
5. rollback/restauração testados;
6. curva de aprendizagem compatível com equipe formativa.

## Recomendação

Para uma próxima etapa, usar OpenTofu para recursos cloud e manter Kustomize/Helm para workloads. OpenTofu fecha a lacuna de provisionamento sem reescrever os manifests. Ansible só seria adicionado se surgirem hosts/rotinas que não pertençam ao modelo declarativo de cloud/Kubernetes.

Antes disso, definir backend remoto, estratégia de lock, identidade do CI, convenção de módulos, scanner (`trivy config`/Checkov), plano em pull request e política de destruição. Não executar `apply` de produção a partir de laptop pessoal.

## Experimento proposto

Criar um módulo mínimo para rede + cluster gerenciado em sandbox, executar `fmt`, `validate` e `plan` no CI e medir: tempo de criação, drift detectado, complexidade do estado e custo mensal. Nenhum recurso deve ser criado sem orçamento e autorização.

# Security policy

Este repositório é um laboratório local. Não o exponha diretamente à internet nem reutilize suas credenciais didáticas.

## Segredos

- Nunca versione `.env`, tokens, chaves, certificados ou dumps de banco.
- Os valores em `.env.example` e no overlay `kind/local` são públicos, locais e descartáveis.
- Em outro ambiente, injete segredos por um gerenciador externo e rotacione-os regularmente.

## Falhas controladas

`LAB_MODE=true` habilita as fontes `slow`, `flaky` e `force-error`. Esse modo deve permanecer desabilitado fora do laboratório.

## Relato responsável

Não publique credenciais ou dados sensíveis em uma issue. Envie uma descrição privada ao mantenedor do repositório, incluindo impacto, forma de reprodução sem dados reais e versão afetada.

## Dependências

Dependabot propõe atualizações semanais e o CI executa Trivy para vulnerabilidades, segredos e configurações inseguras. Um alerta não deve ser silenciado sem justificativa registrada.

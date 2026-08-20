# Verificação de backup falhou

Severidade: crítica quando o backup é necessário para atender RPO/RTO; no laboratório, trate como bloqueio da evidência.

## Tipos de falha

- `pg_dump` não criou arquivo válido;
- criação/restauração do banco temporário falhou;
- schema/migrações não restauraram;
- contagem de importações divergiu;
- espaço ou permissão insuficiente.

## Procedimento

1. Preserve stdout/stderr, horário, versão do PostgreSQL e tamanho/checksum do dump; não publique o dump.
2. Confirme saúde e espaço antes de repetir.
3. Verifique se origem e ferramenta usam versões compatíveis.
4. Garanta que o banco temporário começa com `lab_restore_check_`; não mude o script para apontar a produção.
5. Corrija causa em ambiente isolado e repita `make backup`.

## Validação

O comando deve restaurar sem erro, comparar importações e migrações e produzir `artifacts/backup-verification.json` com `result: pass`. Para produção, contagem é insuficiente: incluir checksums/amostragem, permissões, extensões, RPO/RTO e teste de aplicação.

## Escalar

Escale imediatamente se não houver backup restaurável dentro do RPO, se o dump puder conter dados de terceiros fora do local seguro ou se qualquer ação envolver sobrescrever um banco existente.

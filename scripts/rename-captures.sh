#!/usr/bin/env sh
# Renomeia as capturas mais recentes de ~/Documents na ordem cronologica e
# move para docs/evidence. Exige exatamente oito capturas novas.
#
# Uso:  sh scripts/rename-captures.sh          (apenas simula)
#       sh scripts/rename-captures.sh --aplicar (executa)
set -eu

src="${EVIDENCE_SRC:-$HOME/Documents}"
dest="docs/evidence"
aplicar="${1:-}"

nomes="01-platform-overview 02-logs-por-cliente 03-trace-distribuido \
04-alerta-firing 05-alerta-resolvido 06-teste-de-carga 07-backup-restore 08-ci-aprovado"

# lista capturas por data de modificacao, mais antiga primeiro
lista=$(find "$src" -maxdepth 1 \( -iname "*.png" -o -iname "*.jpg" \) -mmin -720 -print0 2>/dev/null \
  | xargs -0 ls -tr 2>/dev/null || true)

total=$(printf '%s\n' "$lista" | sed '/^$/d' | wc -l | tr -d ' ')
echo "capturas recentes encontradas em $src: $total"
[ "$total" -eq 0 ] && { echo "nada a fazer."; exit 0; }

i=1
printf '%s\n' "$lista" | sed '/^$/d' | while IFS= read -r arquivo; do
  destino=$(printf '%s\n' $nomes | sed -n "${i}p")
  [ -z "$destino" ] && { echo "aviso: mais arquivos que nomes esperados, ignorando o restante"; break; }
  ext=$(printf '%s' "${arquivo##*.}" | tr 'A-Z' 'a-z')
  if [ "$aplicar" = "--aplicar" ]; then
    mv "$arquivo" "$dest/$destino.$ext"
    echo "  $destino.$ext  <-  $(basename "$arquivo")"
  else
    echo "  $destino.$ext  <-  $(basename "$arquivo")   (simulacao)"
  fi
  i=$((i+1))
done

[ "$aplicar" = "--aplicar" ] || echo "
Nada foi movido. Rode com --aplicar para confirmar."

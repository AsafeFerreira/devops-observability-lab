#!/usr/bin/env sh
# Move capturas do Desktop para docs/evidence e reporta o que ainda falta.
# Uso: sh scripts/publish-evidence.sh
set -eu

dest="docs/evidence"
src="${EVIDENCE_SRC:-$HOME/Desktop}"

esperados="01-platform-overview 02-logs-por-cliente 03-trace-distribuido \
04-alerta-firing 05-alerta-resolvido 06-teste-de-carga 07-backup-restore 08-ci-aprovado"

movidos=0
for nome in $esperados; do
  for ext in png jpg jpeg PNG JPG; do
    if [ -f "$src/$nome.$ext" ]; then
      mv "$src/$nome.$ext" "$dest/$nome.$ext"
      echo "movido: $nome.$ext"
      movidos=$((movidos+1))
      break
    fi
  done
done

echo
echo "presentes em $dest:"
faltam=0
for nome in $esperados; do
  if ls "$dest/$nome".* >/dev/null 2>&1; then
    echo "  ok      $nome"
  else
    echo "  falta   $nome"
    faltam=$((faltam+1))
  fi
done

echo
echo "$movidos movida(s) nesta execucao, $faltam pendente(s)."

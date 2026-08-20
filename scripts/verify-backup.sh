#!/usr/bin/env sh
set -eu

postgres_user="${POSTGRES_USER:-observability}"
postgres_db="${POSTGRES_DB:-observability_lab}"
restore_db="lab_restore_check_$(date +%s)_$$"
backup_file="artifacts/backups/${postgres_db}-$(date -u +%Y%m%dT%H%M%SZ).dump"
report_file="artifacts/backup-verification.json"

case "$restore_db" in
  lab_restore_check_[0-9]*) ;;
  *) printf '%s\n' "Unsafe restore database name: ${restore_db}" >&2; exit 1 ;;
esac

etapa() {
  printf '  %s %s\n' "$1" "$2"
}

printf '\n  Verificacao de backup e restauracao do PostgreSQL\n'
printf '  %s\n\n' "------------------------------------------------------------"
etapa "banco de origem   " "$postgres_db"
etapa "banco temporario  " "$restore_db"
printf '\n'

cleanup() {
  docker compose exec -T postgres dropdb --if-exists -U "$postgres_user" "$restore_db" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

mkdir -p artifacts/backups
etapa "[1/5]" "pg_dump do banco de desenvolvimento"
docker compose exec -T postgres pg_dump -U "$postgres_user" -d "$postgres_db" --format=custom >"$backup_file"
etapa "[2/5]" "criando banco temporario isolado"
docker compose exec -T postgres createdb -U "$postgres_user" "$restore_db"
etapa "[3/5]" "restaurando o dump no banco temporario"
docker compose exec -T postgres pg_restore -U "$postgres_user" -d "$restore_db" --no-owner --no-privileges <"$backup_file"

etapa "[4/5]" "comparando contagens entre origem e restauracao"
source_count="$(docker compose exec -T postgres psql -U "$postgres_user" -d "$postgres_db" -Atc 'SELECT count(*) FROM imports')"
restore_count="$(docker compose exec -T postgres psql -U "$postgres_user" -d "$restore_db" -Atc 'SELECT count(*) FROM imports')"
source_migrations="$(docker compose exec -T postgres psql -U "$postgres_user" -d "$postgres_db" -Atc 'SELECT count(*) FROM schema_migrations')"
restore_migrations="$(docker compose exec -T postgres psql -U "$postgres_user" -d "$restore_db" -Atc 'SELECT count(*) FROM schema_migrations')"

if [ "$source_count" != "$restore_count" ] || [ "$source_migrations" != "$restore_migrations" ]; then
  printf '%s\n' "Backup verification failed: row counts differ." >&2
  exit 1
fi

etapa "[5/5]" "removendo apenas os recursos temporarios criados"
printf '\n'
printf '  imports     origem %-8s restaurado %-8s\n' "$source_count" "$restore_count"
printf '  migracoes   origem %-8s restaurado %-8s\n' "$source_migrations" "$restore_migrations"
printf '\n'

printf '{"verifiedAt":"%s","backup":"%s","imports":%s,"migrations":%s,"result":"pass"}\n' \
  "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$backup_file" "$restore_count" "$restore_migrations" >"$report_file"
printf '  RESULTADO: PASS   backup restaurado e verificado (%s imports)\n' "$restore_count"
printf '\n  dump      %s\n' "$backup_file"
printf '  relatorio %s\n\n' "$report_file"

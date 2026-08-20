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

cleanup() {
  docker compose exec -T postgres dropdb --if-exists -U "$postgres_user" "$restore_db" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

mkdir -p artifacts/backups
docker compose exec -T postgres pg_dump -U "$postgres_user" -d "$postgres_db" --format=custom >"$backup_file"
docker compose exec -T postgres createdb -U "$postgres_user" "$restore_db"
docker compose exec -T postgres pg_restore -U "$postgres_user" -d "$restore_db" --no-owner --no-privileges <"$backup_file"

source_count="$(docker compose exec -T postgres psql -U "$postgres_user" -d "$postgres_db" -Atc 'SELECT count(*) FROM imports')"
restore_count="$(docker compose exec -T postgres psql -U "$postgres_user" -d "$restore_db" -Atc 'SELECT count(*) FROM imports')"
source_migrations="$(docker compose exec -T postgres psql -U "$postgres_user" -d "$postgres_db" -Atc 'SELECT count(*) FROM schema_migrations')"
restore_migrations="$(docker compose exec -T postgres psql -U "$postgres_user" -d "$restore_db" -Atc 'SELECT count(*) FROM schema_migrations')"

if [ "$source_count" != "$restore_count" ] || [ "$source_migrations" != "$restore_migrations" ]; then
  printf '%s\n' "Backup verification failed: row counts differ." >&2
  exit 1
fi

printf '{"verifiedAt":"%s","backup":"%s","imports":%s,"migrations":%s,"result":"pass"}\n' \
  "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$backup_file" "$restore_count" "$restore_migrations" >"$report_file"
printf 'OK   backup restored and verified (%s imports)\n' "$restore_count"
printf 'Evidence: %s and %s\n' "$backup_file" "$report_file"

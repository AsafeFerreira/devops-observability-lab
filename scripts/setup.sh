#!/usr/bin/env sh
set -eu

if [ ! -f .env ]; then
  cp .env.example .env
  printf '%s\n' "Created .env from the documented local-only example."
else
  printf '%s\n' ".env already exists; it was not overwritten."
fi
mkdir -p artifacts/backups

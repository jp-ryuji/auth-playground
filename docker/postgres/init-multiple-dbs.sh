#!/usr/bin/env bash
# Creates one database per name in POSTGRES_MULTIPLE_DATABASES (comma-separated).
# Runs once on first postgresd init.
set -euo pipefail

if [[ -z "${POSTGRES_MULTIPLE_DATABASES:-}" ]]; then
  echo "POSTGRES_MULTIPLE_DATABASES not set; skipping"
  exit 0
fi

for db in ${POSTGRES_MULTIPLE_DATABASES//,/ }; do
  echo "Creating database '$db'"
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-SQL
    CREATE DATABASE "$db";
    GRANT ALL PRIVILEGES ON DATABASE "$db" TO "$POSTGRES_USER";
SQL
done

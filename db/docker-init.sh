#!/bin/bash
set -euo pipefail
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
  \i /docker-entrypoint-initdb.d/migrations/001_init.sql
  \i /docker-entrypoint-initdb.d/migrations/002_timescale.sql
  \i /docker-entrypoint-initdb.d/migrations/003_auth_lookup.sql
  \i /docker-entrypoint-initdb.d/seed/001_demo_tenants.sql
  \i /docker-entrypoint-initdb.d/migrations/004_rls_roles.sql
EOSQL

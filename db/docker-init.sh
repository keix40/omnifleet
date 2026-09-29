#!/bin/bash
set -euo pipefail

APP_PASSWORD="${OMNIFLEET_APP_PASSWORD:-omnifleet_app}"

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
  \i /docker-entrypoint-initdb.d/migrations/001_init.sql
  \i /docker-entrypoint-initdb.d/migrations/002_timescale.sql
  \i /docker-entrypoint-initdb.d/migrations/003_auth_lookup.sql
  \i /docker-entrypoint-initdb.d/seed/001_demo_tenants.sql
  \i /docker-entrypoint-initdb.d/seed/002_admin_users.sql
  \i /docker-entrypoint-initdb.d/migrations/004_rls_roles.sql
  \i /docker-entrypoint-initdb.d/migrations/005_rls_with_check.sql
  \i /docker-entrypoint-initdb.d/migrations/006_composite_tenant_fks.sql
  \i /docker-entrypoint-initdb.d/migrations/007_auth_lookup_tenant.sql
  \i /docker-entrypoint-initdb.d/migrations/008_outbox_ws_tickets.sql
  \i /docker-entrypoint-initdb.d/migrations/009_dispatch_jobs.sql
  \i /docker-entrypoint-initdb.d/migrations/010_billing_notifications.sql
EOSQL

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
  -c "ALTER ROLE omnifleet_app PASSWORD '${APP_PASSWORD}';"

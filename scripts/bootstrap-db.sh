#!/usr/bin/env bash
set -euo pipefail

PGHOST="${PGHOST:-localhost}"
PGUSER="${PGUSER:-omnifleet}"
PGPASSWORD="${PGPASSWORD:-omnifleet}"
PGDATABASE="${PGDATABASE:-omnifleet}"
APP_PASSWORD="${OMNIFLEET_APP_PASSWORD:-omnifleet_app}"
export PGPASSWORD

run_migration() {
  psql -h "$PGHOST" -U "$PGUSER" -d "$PGDATABASE" -v ON_ERROR_STOP=1 -f "$1"
}

run_migration db/migrations/001_init.sql
run_migration db/migrations/002_timescale.sql
run_migration db/migrations/003_auth_lookup.sql
run_migration db/seed/001_demo_tenants.sql
run_migration db/migrations/004_rls_roles.sql
run_migration db/migrations/005_rls_with_check.sql
run_migration db/migrations/006_composite_tenant_fks.sql
run_migration db/migrations/007_auth_lookup_tenant.sql
run_migration db/migrations/008_outbox_ws_tickets.sql

psql -h "$PGHOST" -U "$PGUSER" -d "$PGDATABASE" -v ON_ERROR_STOP=1 \
  -c "ALTER ROLE omnifleet_app PASSWORD '${APP_PASSWORD}';"

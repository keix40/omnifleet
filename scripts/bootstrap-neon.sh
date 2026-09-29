#!/usr/bin/env bash
# Bootstrap for Neon and other managed Postgres without superuser.
# Uses DATABASE_URL (or PG* vars). Does not require Timescale or custom roles.
set -euo pipefail

PGHOST="${PGHOST:-localhost}"
PGUSER="${PGUSER:-omnifleet}"
PGPASSWORD="${PGPASSWORD:-omnifleet}"
PGDATABASE="${PGDATABASE:-omnifleet}"
APP_PASSWORD="${OMNIFLEET_APP_PASSWORD:-omnifleet_app}"
export PGPASSWORD

if [[ -n "${DATABASE_URL:-}" ]]; then
  psql "$DATABASE_URL" -v ON_ERROR_STOP=0 -c "SELECT 1" >/dev/null 2>&1 || true
  PSQL=(psql "$DATABASE_URL" -v ON_ERROR_STOP=1)
else
  PSQL=(psql -h "$PGHOST" -U "$PGUSER" -d "$PGDATABASE" -v ON_ERROR_STOP=1)
fi

run_migration() {
  "${PSQL[@]}" -f "$1"
}

echo "==> PostGIS"
"${PSQL[@]}" -c "CREATE EXTENSION IF NOT EXISTS postgis;"

run_migration db/migrations/001_init.sql

echo "==> Timescale (optional)"
if "${PSQL[@]}" -c "CREATE EXTENSION IF NOT EXISTS timescaledb;" 2>/dev/null; then
  run_migration db/migrations/002_timescale.sql
else
  echo "Timescale not available; using optional hypertable migration"
  run_migration db/migrations/012_timescale_hypertable_optional.sql
fi

run_migration db/migrations/003_auth_lookup.sql
run_migration db/seed/001_demo_tenants.sql
run_migration db/seed/002_admin_users.sql

echo "==> RLS roles (fallback to single-role Neon mode)"
set +e
run_migration db/migrations/004_rls_roles.sql
rc=$?
set -e
if [[ "$rc" -ne 0 ]]; then
  echo "004 skipped; applying Neon single-role RLS"
  run_migration db/migrations/013_neon_single_role_rls.sql
fi

run_migration db/migrations/005_rls_with_check.sql
run_migration db/migrations/006_composite_tenant_fks.sql
run_migration db/migrations/007_auth_lookup_tenant.sql
run_migration db/migrations/008_outbox_ws_tickets.sql
run_migration db/migrations/009_dispatch_jobs.sql
run_migration db/migrations/010_billing_notifications.sql

if "${PSQL[@]}" -tAc "SELECT 1 FROM pg_roles WHERE rolname='omnifleet_app'" 2>/dev/null | grep -q 1; then
  "${PSQL[@]}" -c "ALTER ROLE omnifleet_app PASSWORD '${APP_PASSWORD}';"
fi

echo "Neon bootstrap complete."

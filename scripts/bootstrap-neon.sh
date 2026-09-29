#!/usr/bin/env bash
# Bootstrap for Neon and other managed Postgres without superuser.
# Safe to run twice. Always creates omnifleet_app (RLS-enforced app login).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PGHOST="${PGHOST:-localhost}"
PGUSER="${PGUSER:-omnifleet}"
PGPASSWORD="${PGPASSWORD:-omnifleet}"
PGDATABASE="${PGDATABASE:-omnifleet}"
APP_PASSWORD="${OMNIFLEET_APP_PASSWORD:-omnifleet_app}"
export PGPASSWORD

if [[ -n "${DATABASE_URL:-}" ]]; then
  PSQL=(psql "$DATABASE_URL" -v ON_ERROR_STOP=1)
else
  PSQL=(psql -h "$PGHOST" -U "$PGUSER" -d "$PGDATABASE" -v ON_ERROR_STOP=1)
fi

run_migration() {
  "${PSQL[@]}" -f "$1"
}

ensure_omnifleet_app_role() {
  "${PSQL[@]}" <<SQL
DO \$\$
BEGIN
    CREATE ROLE omnifleet_app LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
EXCEPTION
    WHEN duplicate_object THEN NULL;
END
\$\$;
ALTER ROLE omnifleet_app PASSWORD '${APP_PASSWORD}';
SQL
}

finalize_app_grants() {
  "${PSQL[@]}" <<'SQL'
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'omnifleet_app') THEN
        RETURN;
    END IF;
    GRANT USAGE ON SCHEMA public TO omnifleet_app;
    GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO omnifleet_app;
    GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO omnifleet_app;
    GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO omnifleet_app;
END
$$;
SQL
}

print_app_database_url() {
  if [[ -z "${DATABASE_URL:-}" ]]; then
    echo "Set DATABASE_URL on Render to the omnifleet_app connection string, for example:"
    echo "  postgres://omnifleet_app:${APP_PASSWORD}@${PGHOST}:5432/${PGDATABASE}?sslmode=disable"
    return
  fi
  python3 - "$DATABASE_URL" "$APP_PASSWORD" <<'PY'
import sys
from urllib.parse import urlparse, urlunparse, quote

url = sys.argv[1]
password = sys.argv[2]
p = urlparse(url)
user = quote("omnifleet_app", safe="")
pw = quote(password, safe="")
host = p.hostname or ""
port = f":{p.port}" if p.port else ""
netloc = f"{user}:{pw}@{host}{port}"
print(urlunparse((p.scheme, netloc, p.path, p.params, p.query, p.fragment)))
PY
}

echo "==> PostGIS"
"${PSQL[@]}" -c "CREATE EXTENSION IF NOT EXISTS postgis;"

echo "==> omnifleet_app role (RLS-enforced login)"
ensure_omnifleet_app_role

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
chmod +x "$ROOT/scripts/seed-demo-users.sh"
"$ROOT/scripts/seed-demo-users.sh"

run_migration db/migrations/004_rls_roles.sql
run_migration db/migrations/005_rls_with_check.sql
run_migration db/migrations/006_composite_tenant_fks.sql
run_migration db/migrations/007_auth_lookup_tenant.sql
run_migration db/migrations/008_outbox_ws_tickets.sql
run_migration db/migrations/009_dispatch_jobs.sql
run_migration db/migrations/010_billing_notifications.sql

ensure_omnifleet_app_role
finalize_app_grants

echo ""
echo "Neon bootstrap complete."
echo "Use this DATABASE_URL for the API (omnifleet_app, RLS enforced):"
print_app_database_url

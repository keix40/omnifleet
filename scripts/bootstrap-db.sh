#!/usr/bin/env bash
set -euo pipefail

PGHOST="${PGHOST:-localhost}"
PGUSER="${PGUSER:-omnifleet}"
PGPASSWORD="${PGPASSWORD:-omnifleet}"
PGDATABASE="${PGDATABASE:-omnifleet}"
export PGPASSWORD

psql -h "$PGHOST" -U "$PGUSER" -d "$PGDATABASE" -v ON_ERROR_STOP=1 -f db/migrations/001_init.sql
psql -h "$PGHOST" -U "$PGUSER" -d "$PGDATABASE" -v ON_ERROR_STOP=1 -f db/migrations/002_timescale.sql
psql -h "$PGHOST" -U "$PGUSER" -d "$PGDATABASE" -v ON_ERROR_STOP=1 -f db/migrations/003_auth_lookup.sql
psql -h "$PGHOST" -U "$PGUSER" -d "$PGDATABASE" -v ON_ERROR_STOP=1 -f db/seed/001_demo_tenants.sql
psql -h "$PGHOST" -U "$PGUSER" -d "$PGDATABASE" -v ON_ERROR_STOP=1 -f db/migrations/004_rls_roles.sql

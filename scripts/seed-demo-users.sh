#!/usr/bin/env bash
# Inserts demo users with SEED_DEMO_PASSWORD (default: demo-password-change-me).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SEED_DEMO_PASSWORD="${SEED_DEMO_PASSWORD:-demo-password-change-me}"
export SEED_DEMO_PASSWORD

HASH="$(cd "$ROOT/scripts/seedhash" && go run .)"

if [[ -n "${DATABASE_URL:-}" ]]; then
  PSQL=(psql "$DATABASE_URL" -v ON_ERROR_STOP=1)
else
  PGHOST="${PGHOST:-localhost}"
  PGUSER="${PGUSER:-omnifleet}"
  PGDATABASE="${PGDATABASE:-omnifleet}"
  export PGPASSWORD="${PGPASSWORD:-omnifleet}"
  PSQL=(psql -h "$PGHOST" -U "$PGUSER" -d "$PGDATABASE" -v ON_ERROR_STOP=1)
fi

SQL="$(cat <<EOF
INSERT INTO users (id, tenant_id, email, password_hash, role) VALUES
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111',
     'dispatcher@acme.test', '${HASH}', 'dispatcher'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '11111111-1111-1111-1111-111111111111',
     'driver@acme.test', '${HASH}', 'driver'),
    ('cccccccc-cccc-cccc-cccc-cccccccccccc', '22222222-2222-2222-2222-222222222222',
     'dispatcher@globex.test', '${HASH}', 'dispatcher'),
    ('dddddddd-dddd-dddd-dddd-dddddddddddd', '22222222-2222-2222-2222-222222222222',
     'driver@globex.test', '${HASH}', 'driver'),
    ('11111111-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111',
     'admin@acme.test', '${HASH}', 'admin')
ON CONFLICT (tenant_id, email) DO UPDATE SET password_hash = EXCLUDED.password_hash;
EOF
)"

"${PSQL[@]}" -c "$SQL"

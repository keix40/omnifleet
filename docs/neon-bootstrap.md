# Neon-compatible database bootstrap

OmniFleet can run on [Neon](https://neon.tech) and other managed Postgres offerings that do not grant superuser.

## Requirements

- **PostGIS** — enable in Neon dashboard (`CREATE EXTENSION postgis`).
- **TimescaleDB** — optional. When unavailable, `gps_positions` stays a normal table (no compression or paid Timescale features).
- **Roles** — if `CREATE ROLE` is not allowed, use `013_neon_single_role_rls.sql` via `scripts/bootstrap-neon.sh`.

## Environment

| Variable | Purpose |
|----------|---------|
| `DATABASE_URL` | Pooled app connection (RLS enforced via `app.tenant_id`) |
| `OMNIFLEET_APP_PASSWORD` | Password for `omnifleet_app` when role bootstrap succeeds |
| `MIGRATION_DATABASE_URL` | Optional direct (non-pooler) URL for one-time bootstrap |

## Bootstrap

```bash
export DATABASE_URL="postgres://user:pass@ep-....neon.tech/neondb?sslmode=require"
export OMNIFLEET_APP_PASSWORD="choose-a-strong-password"
chmod +x scripts/bootstrap-neon.sh
./scripts/bootstrap-neon.sh
```

## Graceful fallbacks

| Feature | Fallback |
|---------|----------|
| Timescale hypertable | Regular `gps_positions` table |
| `omnifleet_app` role | Current migration user + RLS |
| Timescale compression | Not used (Apache-2 slice only) |

## Verify RLS

Use the same `pkg/db` RLS tests with `APP_DATABASE_URL` set to your app role connection string.

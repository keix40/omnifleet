# Neon-compatible database bootstrap

OmniFleet can run on [Neon](https://neon.tech) and other managed Postgres offerings that do not grant superuser.

## Requirements

- **PostGIS** — enable in Neon dashboard (`CREATE EXTENSION postgis`).
- **TimescaleDB** — optional. When unavailable, `gps_positions` stays a normal table (no compression or paid Timescale features).

## Environment (bootstrap only)

| Variable | Purpose |
|----------|---------|
| `DATABASE_URL` | Migration/owner connection for running the script |
| `OMNIFLEET_APP_PASSWORD` | Login password for the `omnifleet_app` role (not read by the API at runtime) |
| `SEED_DEMO_PASSWORD` | Demo tenant user password (default `demo-password-change-me`) |

## Bootstrap

```bash
export DATABASE_URL="postgres://owner:pass@ep-....neon.tech/neondb?sslmode=require"
export OMNIFLEET_APP_PASSWORD="choose-a-strong-password"
export SEED_DEMO_PASSWORD="demo-password-change-me"   # optional
chmod +x scripts/bootstrap-neon.sh
./scripts/bootstrap-neon.sh
```

The script is **idempotent** (safe to run twice). It always creates a non-owner, non-`BYPASSRLS` **`omnifleet_app`** login and prints the **`DATABASE_URL`** to configure on Render.

## API connection

Point Render `DATABASE_URL` at the **`omnifleet_app`** connection string from the script output so row-level security applies. Connecting as the Neon owner bypasses RLS.

## Graceful fallbacks

| Feature | Fallback |
|---------|----------|
| Timescale hypertable | Regular `gps_positions` table |
| `omnifleet_owner` BYPASSRLS | Tables stay owned by migration user; `FORCE RLS` + `omnifleet_app` still enforce tenancy |

## Verify RLS

Use the same `pkg/db` RLS tests with `APP_DATABASE_URL` set to your app role connection string.

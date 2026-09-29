# ADR 0002: Shared Postgres with row-level security

## Status

Accepted

## Context

Enterprise SaaS fleets require strong tenant isolation without operating hundreds of databases.

## Decision

Use a **shared Postgres** cluster (Aurora in production) with `tenant_id` on all tenant-owned tables and **RLS policies** comparing `tenant_id` to `current_setting('app.tenant_id')`.

## Rationale

- Centralizes schema migrations and reporting.
- RLS provides defense-in-depth even if application code regresses.
- PostGIS + TimescaleDB extensions stay co-located for geospatial and GPS history.

## Consequences

- Every request transaction must set `app.tenant_id` (see `pkg/db/WithTenant`).
- Auth login uses `SECURITY DEFINER` function `auth_lookup_user` before tenant context exists.
- Integration tests must prove cross-tenant reads fail (`pkg/db/rls_test.go`).

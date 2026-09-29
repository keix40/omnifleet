-- Neon / managed Postgres without superuser: skip omnifleet_owner/omnifleet_app roles.
-- The migration user owns tables and sets tenant context via set_config('app.tenant_id', ...).

DO $$
DECLARE
    r record;
BEGIN
    FOR r IN
        SELECT c.relname
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = 'public'
          AND c.relkind = 'r'
          AND c.relname NOT IN ('tenants', 'billing_plans', 'schema_migrations')
    LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', r.relname);
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', r.relname);
    END LOOP;
EXCEPTION
    WHEN insufficient_privilege THEN
        RAISE NOTICE 'omnifleet: could not FORCE RLS on all tables (insufficient privilege)';
END
$$;

GRANT USAGE ON SCHEMA public TO CURRENT_USER;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO CURRENT_USER;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO CURRENT_USER;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO CURRENT_USER;

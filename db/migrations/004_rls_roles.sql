-- Application DB role: not superuser, does not own tables, subject to FORCE RLS.

DO $$
BEGIN
    CREATE ROLE omnifleet_owner NOLOGIN;
EXCEPTION
    WHEN duplicate_object THEN NULL;
END
$$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_roles WHERE rolname = 'omnifleet_owner' AND NOT rolbypassrls
    ) THEN
        ALTER ROLE omnifleet_owner BYPASSRLS;
    END IF;
EXCEPTION
    WHEN insufficient_privilege THEN
        RAISE NOTICE 'omnifleet: skipping BYPASSRLS on omnifleet_owner (insufficient privilege)';
END
$$;

-- Password is set by db/docker-init.sh or scripts/bootstrap-db.sh from OMNIFLEET_APP_PASSWORD.
DO $$
BEGIN
    CREATE ROLE omnifleet_app LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
EXCEPTION
    WHEN duplicate_object THEN NULL;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'omnifleet_owner') THEN
        RETURN;
    END IF;
    BEGIN
        EXECUTE format('GRANT omnifleet_owner TO %I', CURRENT_USER);
    EXCEPTION
        WHEN OTHERS THEN
            RAISE NOTICE 'omnifleet: could not grant omnifleet_owner to migration user';
    END;
    BEGIN
        ALTER TABLE tenants OWNER TO omnifleet_owner;
        ALTER TABLE users OWNER TO omnifleet_owner;
        ALTER TABLE vehicles OWNER TO omnifleet_owner;
        ALTER TABLE geofences OWNER TO omnifleet_owner;
        ALTER TABLE vehicle_geofence_state OWNER TO omnifleet_owner;
        ALTER TABLE gps_positions OWNER TO omnifleet_owner;
    EXCEPTION
        WHEN OTHERS THEN
            RAISE NOTICE 'omnifleet: table ownership transfer to omnifleet_owner skipped (%)', SQLERRM;
    END;
END
$$;

ALTER TABLE users FORCE ROW LEVEL SECURITY;
ALTER TABLE vehicles FORCE ROW LEVEL SECURITY;
ALTER TABLE geofences FORCE ROW LEVEL SECURITY;
ALTER TABLE vehicle_geofence_state FORCE ROW LEVEL SECURITY;
ALTER TABLE gps_positions FORCE ROW LEVEL SECURITY;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'omnifleet_app') THEN
        GRANT USAGE ON SCHEMA public TO omnifleet_app;
        GRANT USAGE ON TYPE user_role TO omnifleet_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO omnifleet_app;
        GRANT EXECUTE ON FUNCTION auth_lookup_user(TEXT, TEXT) TO omnifleet_app;
    END IF;
END
$$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'omnifleet_owner')
       AND EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'omnifleet_app') THEN
        ALTER DEFAULT PRIVILEGES FOR ROLE omnifleet_owner IN SCHEMA public
            GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO omnifleet_app;
    END IF;
END
$$;

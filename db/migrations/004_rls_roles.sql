-- Application DB role: not superuser, does not own tables, subject to FORCE RLS.

DO $$
BEGIN
    CREATE ROLE omnifleet_owner NOLOGIN BYPASSRLS;
EXCEPTION
    WHEN duplicate_object THEN NULL;
END
$$;

ALTER ROLE omnifleet_owner BYPASSRLS;

DO $$
BEGIN
    CREATE ROLE omnifleet_app LOGIN PASSWORD 'omnifleet_app';
EXCEPTION
    WHEN duplicate_object THEN NULL;
END
$$;

ALTER TABLE users OWNER TO omnifleet_owner;
ALTER TABLE vehicles OWNER TO omnifleet_owner;
ALTER TABLE geofences OWNER TO omnifleet_owner;
ALTER TABLE vehicle_geofence_state OWNER TO omnifleet_owner;
ALTER TABLE gps_positions OWNER TO omnifleet_owner;

ALTER TABLE users FORCE ROW LEVEL SECURITY;
ALTER TABLE vehicles FORCE ROW LEVEL SECURITY;
ALTER TABLE geofences FORCE ROW LEVEL SECURITY;
ALTER TABLE vehicle_geofence_state FORCE ROW LEVEL SECURITY;
ALTER TABLE gps_positions FORCE ROW LEVEL SECURITY;

ALTER FUNCTION auth_lookup_user(TEXT) OWNER TO omnifleet_owner;

GRANT USAGE ON SCHEMA public TO omnifleet_app;
GRANT USAGE ON TYPE user_role TO omnifleet_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO omnifleet_app;
GRANT EXECUTE ON FUNCTION auth_lookup_user(TEXT) TO omnifleet_app;

ALTER DEFAULT PRIVILEGES FOR ROLE omnifleet_owner IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO omnifleet_app;

-- Dispatch jobs with PostGIS pickup/dropoff and tenant-scoped composite FKs.

ALTER TABLE users
    ADD CONSTRAINT users_tenant_id_id_key UNIQUE (tenant_id, id);

CREATE TYPE job_status AS ENUM (
    'created',
    'assigned',
    'en_route',
    'picked_up',
    'delivered',
    'cancelled'
);

CREATE TABLE dispatch_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    status job_status NOT NULL DEFAULT 'created',
    pickup GEOGRAPHY(POINT, 4326) NOT NULL,
    dropoff GEOGRAPHY(POINT, 4326) NOT NULL,
    pickup_label TEXT,
    dropoff_label TEXT,
    vehicle_id UUID,
    driver_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT dispatch_jobs_tenant_vehicle_fkey
        FOREIGN KEY (tenant_id, vehicle_id) REFERENCES vehicles (tenant_id, id) ON DELETE SET NULL,
    CONSTRAINT dispatch_jobs_tenant_driver_fkey
        FOREIGN KEY (tenant_id, driver_id) REFERENCES users (tenant_id, id) ON DELETE SET NULL
);

CREATE INDEX idx_dispatch_jobs_tenant_status ON dispatch_jobs (tenant_id, status, updated_at DESC);
CREATE INDEX idx_dispatch_jobs_pickup ON dispatch_jobs USING GIST (pickup);

ALTER TABLE dispatch_jobs ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_dispatch_jobs ON dispatch_jobs
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

ALTER TABLE dispatch_jobs FORCE ROW LEVEL SECURITY;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'omnifleet_owner') THEN
        BEGIN
            ALTER TABLE dispatch_jobs OWNER TO omnifleet_owner;
        EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'omnifleet: dispatch_jobs owner transfer skipped (%)', SQLERRM;
        END;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'omnifleet_app') THEN
        GRANT USAGE ON TYPE job_status TO omnifleet_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON dispatch_jobs TO omnifleet_app;
    END IF;
END
$$;

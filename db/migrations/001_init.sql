-- OmniFleet core schema: shared DB multi-tenancy with row-level security.

CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TYPE user_role AS ENUM ('admin', 'dispatcher', 'driver', 'customer');

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role user_role NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, email)
);

CREATE TABLE vehicles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    label TEXT NOT NULL,
    license_plate TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE geofences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    boundary GEOGRAPHY(POLYGON, 4326) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE vehicle_geofence_state (
    tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    vehicle_id UUID NOT NULL REFERENCES vehicles (id) ON DELETE CASCADE,
    geofence_id UUID NOT NULL REFERENCES geofences (id) ON DELETE CASCADE,
    inside BOOLEAN NOT NULL DEFAULT false,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, vehicle_id, geofence_id)
);

CREATE TABLE gps_positions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    vehicle_id UUID NOT NULL REFERENCES vehicles (id) ON DELETE CASCADE,
    driver_id UUID REFERENCES users (id),
    location GEOGRAPHY(POINT, 4326) NOT NULL,
    speed_mps DOUBLE PRECISION,
    heading_deg DOUBLE PRECISION,
    recorded_at TIMESTAMPTZ NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_gps_positions_tenant_vehicle_time
    ON gps_positions (tenant_id, vehicle_id, recorded_at DESC);

CREATE INDEX idx_geofences_boundary ON geofences USING GIST (boundary);

ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE vehicles ENABLE ROW LEVEL SECURITY;
ALTER TABLE geofences ENABLE ROW LEVEL SECURITY;
ALTER TABLE vehicle_geofence_state ENABLE ROW LEVEL SECURITY;
ALTER TABLE gps_positions ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_users ON users
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_vehicles ON vehicles
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_geofences ON geofences
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_vehicle_geofence_state ON vehicle_geofence_state
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_gps_positions ON gps_positions
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

-- Tenants table is read by platform bootstrap only (no RLS in v1 slice).

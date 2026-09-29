-- Explicit WITH CHECK for INSERT/UPDATE under RLS (vehicle_geofence_state writes from geofencing).

DROP POLICY IF EXISTS tenant_isolation_users ON users;
DROP POLICY IF EXISTS tenant_isolation_vehicles ON vehicles;
DROP POLICY IF EXISTS tenant_isolation_geofences ON geofences;
DROP POLICY IF EXISTS tenant_isolation_vehicle_geofence_state ON vehicle_geofence_state;
DROP POLICY IF EXISTS tenant_isolation_gps_positions ON gps_positions;

CREATE POLICY tenant_isolation_users ON users
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_vehicles ON vehicles
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_geofences ON geofences
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_vehicle_geofence_state ON vehicle_geofence_state
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_gps_positions ON gps_positions
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

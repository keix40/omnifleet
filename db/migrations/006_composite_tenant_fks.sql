-- Composite keys so FK checks respect tenant_id (RLS does not apply to FK validation).

ALTER TABLE vehicles
    ADD CONSTRAINT vehicles_tenant_id_id_key UNIQUE (tenant_id, id);

ALTER TABLE geofences
    ADD CONSTRAINT geofences_tenant_id_id_key UNIQUE (tenant_id, id);

ALTER TABLE gps_positions
    DROP CONSTRAINT IF EXISTS gps_positions_vehicle_id_fkey;

ALTER TABLE gps_positions
    ADD CONSTRAINT gps_positions_tenant_vehicle_fkey
        FOREIGN KEY (tenant_id, vehicle_id) REFERENCES vehicles (tenant_id, id) ON DELETE CASCADE;

ALTER TABLE vehicle_geofence_state
    DROP CONSTRAINT IF EXISTS vehicle_geofence_state_vehicle_id_fkey;

ALTER TABLE vehicle_geofence_state
    DROP CONSTRAINT IF EXISTS vehicle_geofence_state_geofence_id_fkey;

ALTER TABLE vehicle_geofence_state
    ADD CONSTRAINT vehicle_geofence_state_tenant_vehicle_fkey
        FOREIGN KEY (tenant_id, vehicle_id) REFERENCES vehicles (tenant_id, id) ON DELETE CASCADE;

ALTER TABLE vehicle_geofence_state
    ADD CONSTRAINT vehicle_geofence_state_tenant_geofence_fkey
        FOREIGN KEY (tenant_id, geofence_id) REFERENCES geofences (tenant_id, id) ON DELETE CASCADE;

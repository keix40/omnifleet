-- Demo tenants, vehicles, geofences. Users: scripts/seed-demo-users.sh (SEED_DEMO_PASSWORD).

INSERT INTO tenants (id, slug, name) VALUES
    ('11111111-1111-1111-1111-111111111111', 'acme-logistics', 'Acme Logistics'),
    ('22222222-2222-2222-2222-222222222222', 'globex-freight', 'Globex Freight')
ON CONFLICT (slug) DO NOTHING;

DO $$
BEGIN
    PERFORM set_config('app.tenant_id', '11111111-1111-1111-1111-111111111111', true);
    INSERT INTO vehicles (id, tenant_id, label, license_plate) VALUES
        ('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', '11111111-1111-1111-1111-111111111111', 'Acme Truck 1', 'ACM-001')
    ON CONFLICT (id) DO NOTHING;

    PERFORM set_config('app.tenant_id', '22222222-2222-2222-2222-222222222222', true);
    INSERT INTO vehicles (id, tenant_id, label, license_plate) VALUES
        ('ffffffff-ffff-ffff-ffff-ffffffffffff', '22222222-2222-2222-2222-222222222222', 'Globex Van 1', 'GLX-001')
    ON CONFLICT (id) DO NOTHING;

    PERFORM set_config('app.tenant_id', '11111111-1111-1111-1111-111111111111', true);
    INSERT INTO geofences (id, tenant_id, name, boundary) VALUES
        ('99999999-9999-9999-9999-999999999901', '11111111-1111-1111-1111-111111111111', 'Acme SF Depot',
         ST_GeogFromText('POLYGON((-122.4500 37.7600, -122.4000 37.7600, -122.4000 37.8000, -122.4500 37.8000, -122.4500 37.7600))'))
    ON CONFLICT (id) DO NOTHING;

    PERFORM set_config('app.tenant_id', '22222222-2222-2222-2222-222222222222', true);
    INSERT INTO geofences (id, tenant_id, name, boundary) VALUES
        ('99999999-9999-9999-9999-999999999902', '22222222-2222-2222-2222-222222222222', 'Globex NYC Hub',
         ST_GeogFromText('POLYGON((-74.0060 40.7128, -73.9960 40.7128, -73.9960 40.7228, -74.0060 40.7228, -74.0060 40.7128))'))
    ON CONFLICT (id) DO NOTHING;
END
$$;

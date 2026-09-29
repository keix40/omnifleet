-- Demo tenants, users (password: demo-password-change-me), vehicles, geofences.
-- bcrypt hash for "demo-password-change-me" (cost 10).

INSERT INTO tenants (id, slug, name) VALUES
    ('11111111-1111-1111-1111-111111111111', 'acme-logistics', 'Acme Logistics'),
    ('22222222-2222-2222-2222-222222222222', 'globex-freight', 'Globex Freight')
ON CONFLICT (slug) DO NOTHING;

INSERT INTO users (id, tenant_id, email, password_hash, role) VALUES
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111',
     'dispatcher@acme.test', '$2a$10$/6KXkdwnlrqG12iLRZtYfetWfnSLrbpwu3zJzCKB7j.znJaP9OcoG', 'dispatcher'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '11111111-1111-1111-1111-111111111111',
     'driver@acme.test', '$2a$10$/6KXkdwnlrqG12iLRZtYfetWfnSLrbpwu3zJzCKB7j.znJaP9OcoG', 'driver'),
    ('cccccccc-cccc-cccc-cccc-cccccccccccc', '22222222-2222-2222-2222-222222222222',
     'dispatcher@globex.test', '$2a$10$/6KXkdwnlrqG12iLRZtYfetWfnSLrbpwu3zJzCKB7j.znJaP9OcoG', 'dispatcher'),
    ('dddddddd-dddd-dddd-dddd-dddddddddddd', '22222222-2222-2222-2222-222222222222',
     'driver@globex.test', '$2a$10$/6KXkdwnlrqG12iLRZtYfetWfnSLrbpwu3zJzCKB7j.znJaP9OcoG', 'driver')
ON CONFLICT DO NOTHING;

INSERT INTO vehicles (id, tenant_id, label, license_plate) VALUES
    ('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', '11111111-1111-1111-1111-111111111111', 'Acme Truck 1', 'ACM-001'),
    ('ffffffff-ffff-ffff-ffff-ffffffffffff', '22222222-2222-2222-2222-222222222222', 'Globex Van 1', 'GLX-001')
ON CONFLICT DO NOTHING;

-- Warehouse geofences (~500m boxes) near SF and NYC for demo map movement.
INSERT INTO geofences (id, tenant_id, name, boundary) VALUES
    ('99999999-9999-9999-9999-999999999901', '11111111-1111-1111-1111-111111111111', 'Acme SF Depot',
     ST_GeogFromText('POLYGON((-122.4194 37.7749, -122.4094 37.7749, -122.4094 37.7849, -122.4194 37.7849, -122.4194 37.7749))')),
    ('99999999-9999-9999-9999-999999999902', '22222222-2222-2222-2222-222222222222', 'Globex NYC Hub',
     ST_GeogFromText('POLYGON((-74.0060 40.7128, -73.9960 40.7128, -73.9960 40.7228, -74.0060 40.7228, -74.0060 40.7128))'))
ON CONFLICT DO NOTHING;

-- Docker init only (no Go in postgres image). Bootstrap scripts use scripts/seed-demo-users.sh.
-- Default password: demo-password-change-me

INSERT INTO users (id, tenant_id, email, password_hash, role) VALUES
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111',
     'dispatcher@acme.test', '$2a$10$/6KXkdwnlrqG12iLRZtYfetWfnSLrbpwu3zJzCKB7j.znJaP9OcoG', 'dispatcher'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '11111111-1111-1111-1111-111111111111',
     'driver@acme.test', '$2a$10$/6KXkdwnlrqG12iLRZtYfetWfnSLrbpwu3zJzCKB7j.znJaP9OcoG', 'driver'),
    ('cccccccc-cccc-cccc-cccc-cccccccccccc', '22222222-2222-2222-2222-222222222222',
     'dispatcher@globex.test', '$2a$10$/6KXkdwnlrqG12iLRZtYfetWfnSLrbpwu3zJzCKB7j.znJaP9OcoG', 'dispatcher'),
    ('dddddddd-dddd-dddd-dddd-dddddddddddd', '22222222-2222-2222-2222-222222222222',
     'driver@globex.test', '$2a$10$/6KXkdwnlrqG12iLRZtYfetWfnSLrbpwu3zJzCKB7j.znJaP9OcoG', 'driver'),
    ('11111111-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111',
     'admin@acme.test', '$2a$10$/6KXkdwnlrqG12iLRZtYfetWfnSLrbpwu3zJzCKB7j.znJaP9OcoG', 'admin')
ON CONFLICT (tenant_id, email) DO NOTHING;

INSERT INTO users (id, tenant_id, email, password_hash, role) VALUES
    ('11111111-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111',
     'admin@acme.test', '$2a$10$/6KXkdwnlrqG12iLRZtYfetWfnSLrbpwu3zJzCKB7j.znJaP9OcoG', 'admin')
ON CONFLICT DO NOTHING;

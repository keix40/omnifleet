-- Transactional outbox for reliable NATS publish after DB commit.
-- Single-use WebSocket ticket redemptions shared across gateway replicas.

CREATE TABLE event_outbox (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    subject TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);

CREATE INDEX idx_event_outbox_unpublished ON event_outbox (created_at)
    WHERE published_at IS NULL;

CREATE TABLE ws_ticket_redemptions (
    jti TEXT PRIMARY KEY,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_ws_ticket_redemptions_expires ON ws_ticket_redemptions (expires_at);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'omnifleet_owner') THEN
        BEGIN
            ALTER TABLE event_outbox OWNER TO omnifleet_owner;
            ALTER TABLE ws_ticket_redemptions OWNER TO omnifleet_owner;
        EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'omnifleet: outbox owner transfer skipped (%)', SQLERRM;
        END;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'omnifleet_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON event_outbox TO omnifleet_app;
        GRANT SELECT, INSERT, DELETE ON ws_ticket_redemptions TO omnifleet_app;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'omnifleet_owner')
       AND EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'omnifleet_app') THEN
        ALTER DEFAULT PRIVILEGES FOR ROLE omnifleet_owner IN SCHEMA public
            GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO omnifleet_app;
    END IF;
END
$$;

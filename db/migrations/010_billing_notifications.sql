-- Billing subscriptions, usage metering, Stripe webhook idempotency, notification delivery.

CREATE TABLE billing_plans (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    vehicle_limit INT NOT NULL,
    positions_included_monthly BIGINT NOT NULL,
    stripe_price_id TEXT
);

INSERT INTO billing_plans (id, name, vehicle_limit, positions_included_monthly, stripe_price_id) VALUES
    ('starter', 'Starter', 5, 50000, 'price_starter'),
    ('growth', 'Growth', 25, 500000, 'price_growth'),
    ('enterprise', 'Enterprise', 1000, 50000000, 'price_enterprise')
ON CONFLICT (id) DO NOTHING;

CREATE TABLE billing_subscriptions (
    tenant_id UUID PRIMARY KEY REFERENCES tenants (id) ON DELETE CASCADE,
    plan_id TEXT NOT NULL REFERENCES billing_plans (id),
    status TEXT NOT NULL DEFAULT 'active',
    stripe_customer_id TEXT,
    stripe_subscription_id TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE billing_usage_snapshots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    vehicle_count INT NOT NULL,
    position_count BIGINT NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, period_start)
);

CREATE TABLE stripe_webhook_events (
    idempotency_key TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE notification_preferences (
    tenant_id UUID PRIMARY KEY REFERENCES tenants (id) ON DELETE CASCADE,
    alerts_enabled BOOLEAN NOT NULL DEFAULT true,
    dispatch_enabled BOOLEAN NOT NULL DEFAULT true,
    eta_enabled BOOLEAN NOT NULL DEFAULT true,
    channel TEXT NOT NULL DEFAULT 'log',
    webhook_url TEXT,
    email_to TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE notification_deliveries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    channel TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    delivered_at TIMESTAMPTZ
);

CREATE TABLE notification_dlq (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    channel TEXT NOT NULL,
    payload JSONB NOT NULL,
    error TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE billing_subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE billing_usage_snapshots ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_preferences ENABLE ROW LEVEL SECURITY;
-- Deliveries/DLQ are processed by the notifications worker cross-tenant (no RLS).

CREATE POLICY tenant_isolation_billing_subscriptions ON billing_subscriptions
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_billing_usage ON billing_usage_snapshots
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_notification_preferences ON notification_preferences
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

ALTER TABLE billing_subscriptions FORCE ROW LEVEL SECURITY;
ALTER TABLE billing_usage_snapshots FORCE ROW LEVEL SECURITY;
ALTER TABLE notification_preferences FORCE ROW LEVEL SECURITY;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'omnifleet_owner') THEN
        BEGIN
            ALTER TABLE billing_plans OWNER TO omnifleet_owner;
            ALTER TABLE billing_subscriptions OWNER TO omnifleet_owner;
            ALTER TABLE billing_usage_snapshots OWNER TO omnifleet_owner;
            ALTER TABLE stripe_webhook_events OWNER TO omnifleet_owner;
            ALTER TABLE notification_preferences OWNER TO omnifleet_owner;
            ALTER TABLE notification_deliveries OWNER TO omnifleet_owner;
            ALTER TABLE notification_dlq OWNER TO omnifleet_owner;
        EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'omnifleet: billing/notifications owner transfer skipped (%)', SQLERRM;
        END;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'omnifleet_app') THEN
        GRANT SELECT ON billing_plans TO omnifleet_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON billing_subscriptions TO omnifleet_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON billing_usage_snapshots TO omnifleet_app;
        GRANT SELECT, INSERT ON stripe_webhook_events TO omnifleet_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON notification_preferences TO omnifleet_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON notification_deliveries TO omnifleet_app;
        GRANT SELECT, INSERT ON notification_dlq TO omnifleet_app;
    END IF;
END
$$;

-- Default subscriptions for demo tenants.
INSERT INTO billing_subscriptions (tenant_id, plan_id, status) VALUES
    ('11111111-1111-1111-1111-111111111111', 'growth', 'active'),
    ('22222222-2222-2222-2222-222222222222', 'starter', 'active')
ON CONFLICT (tenant_id) DO NOTHING;

INSERT INTO notification_preferences (tenant_id, channel) VALUES
    ('11111111-1111-1111-1111-111111111111', 'log'),
    ('22222222-2222-2222-2222-222222222222', 'log')
ON CONFLICT (tenant_id) DO NOTHING;

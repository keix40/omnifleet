-- Scope login lookup to a tenant; never expose cross-tenant email matches.

DROP FUNCTION IF EXISTS auth_lookup_user(TEXT);

CREATE OR REPLACE FUNCTION auth_lookup_user(p_email TEXT, p_tenant_slug TEXT)
RETURNS TABLE (
    id UUID,
    tenant_id UUID,
    role user_role,
    password_hash TEXT
)
LANGUAGE sql
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT u.id, u.tenant_id, u.role, u.password_hash
    FROM users u
    INNER JOIN tenants t ON t.id = u.tenant_id
    WHERE u.email = p_email AND t.slug = p_tenant_slug
    LIMIT 1;
$$;

REVOKE ALL ON FUNCTION auth_lookup_user(TEXT, TEXT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auth_lookup_user(TEXT, TEXT) TO omnifleet_app;

ALTER FUNCTION auth_lookup_user(TEXT, TEXT) OWNER TO omnifleet_owner;

-- Login join needs tenant slug resolution under SECURITY DEFINER (omnifleet_owner).
ALTER TABLE tenants OWNER TO omnifleet_owner;
GRANT SELECT ON tenants TO omnifleet_app;

-- Login must resolve users before tenant context exists; SECURITY DEFINER bypasses RLS safely for email lookup only.

CREATE OR REPLACE FUNCTION auth_lookup_user(p_email TEXT)
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
    WHERE u.email = p_email
    LIMIT 1;
$$;

REVOKE ALL ON FUNCTION auth_lookup_user(TEXT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auth_lookup_user(TEXT) TO PUBLIC;

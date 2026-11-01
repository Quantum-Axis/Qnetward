-- Row-level security for workspace-owned data (ADR 0009).
--
-- Requests that act inside a workspace switch to qnetward_tenant for the
-- length of their transaction and set app.workspace_id. That role has no
-- BYPASSRLS and owns nothing, so the policies below always apply to it. The
-- login role keeps its own rights for system work: migrations, retention,
-- backfills, and account deletion.
DO $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'qnetward_tenant') THEN
		CREATE ROLE qnetward_tenant NOLOGIN NOBYPASSRLS;
	END IF;
END
$$;

-- The login role must be able to SET ROLE to it. Since PostgreSQL 16 the
-- creator of a role only gets ADMIN on it, so SET is granted explicitly.
DO $$
BEGIN
	IF NOT pg_has_role(current_user, 'qnetward_tenant', 'SET') THEN
		EXECUTE format('GRANT qnetward_tenant TO %I WITH SET TRUE, INHERIT FALSE', current_user);
	END IF;
END
$$;

GRANT USAGE ON SCHEMA public TO qnetward_tenant;
GRANT SELECT, INSERT, UPDATE, DELETE ON alert_rules TO qnetward_tenant;

ALTER TABLE alert_rules ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS workspace_isolation ON alert_rules;
CREATE POLICY workspace_isolation ON alert_rules
	TO qnetward_tenant
	USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::bigint)
	WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::bigint);

-- Alert rules belong to a workspace, so everyone in it sees and edits the same
-- rules. user_id stays as "created by" and survives the author leaving.
ALTER TABLE alert_rules ADD COLUMN IF NOT EXISTS workspace_id BIGINT REFERENCES workspaces(id) ON DELETE CASCADE;

-- Existing rules move to their author's personal workspace (one per user).
UPDATE alert_rules r
SET workspace_id = w.id
FROM workspaces w
WHERE r.workspace_id IS NULL AND w.owner_user_id = r.user_id;

ALTER TABLE alert_rules ALTER COLUMN workspace_id SET NOT NULL;

ALTER TABLE alert_rules ALTER COLUMN user_id DROP NOT NULL;
ALTER TABLE alert_rules DROP CONSTRAINT IF EXISTS alert_rules_user_id_fkey;
ALTER TABLE alert_rules ADD CONSTRAINT alert_rules_user_id_fkey
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

DROP INDEX IF EXISTS idx_alert_rules_lookup;
CREATE INDEX IF NOT EXISTS idx_alert_rules_workspace_lookup
	ON alert_rules (workspace_id, protocol, target, port, enabled);

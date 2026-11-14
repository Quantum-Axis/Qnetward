-- When each person was last active (updated at most every few minutes as
-- their session is used), so a team can see who still uses the workspace.
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ;

-- A workspace can require two-factor sign-in: members without it can still
-- reach their account settings to turn it on, but not the workspace's data.
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS require_mfa BOOLEAN NOT NULL DEFAULT FALSE;

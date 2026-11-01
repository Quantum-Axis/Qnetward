-- Enough about each session to recognise it in "Password and sessions":
-- the browser that created it and the address it came from.
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS user_agent TEXT NOT NULL DEFAULT '';
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS ip_address TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions (user_id, created_at DESC);

-- Object storage key of the uploaded avatar; empty means "use initials".
ALTER TABLE user_profiles ADD COLUMN IF NOT EXISTS avatar_key TEXT NOT NULL DEFAULT '';

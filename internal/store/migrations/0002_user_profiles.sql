-- Public profile details, kept apart from users so the auth table stays small
-- and every login query doesn't carry profile text.

CREATE TABLE IF NOT EXISTS user_profiles (
	user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
	bio TEXT NOT NULL DEFAULT '',
	pronouns TEXT NOT NULL DEFAULT '',
	website TEXT NOT NULL DEFAULT '',
	-- IANA zone name such as Asia/Dhaka; empty means "use the browser's zone".
	timezone TEXT NOT NULL DEFAULT '',
	social_links JSONB NOT NULL DEFAULT '[]'::jsonb,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

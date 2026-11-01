-- HTTP(S) monitors: the target is a URL, the check is a GET, and HTTPS checks
-- record when the certificate expires.
ALTER TABLE monitors DROP CONSTRAINT IF EXISTS monitors_protocol_check;
ALTER TABLE monitors ADD CONSTRAINT monitors_protocol_check CHECK (protocol IN ('ping', 'tcpping', 'http'));

-- NULL accepts any 2xx or 3xx response.
ALTER TABLE monitors ADD COLUMN IF NOT EXISTS expected_status INTEGER
	CHECK (expected_status BETWEEN 100 AND 599);
ALTER TABLE monitors ADD COLUMN IF NOT EXISTS last_status_code INTEGER;
ALTER TABLE monitors ADD COLUMN IF NOT EXISTS tls_expires_at TIMESTAMPTZ;
-- The expiry we last warned about, so each certificate is warned about once.
ALTER TABLE monitors ADD COLUMN IF NOT EXISTS tls_warned_expiry TIMESTAMPTZ;

ALTER TABLE monitor_checks ADD COLUMN IF NOT EXISTS status_code INTEGER;

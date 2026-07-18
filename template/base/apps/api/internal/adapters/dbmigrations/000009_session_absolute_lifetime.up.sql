ALTER TABLE session_models ADD COLUMN absolute_expires_at timestamptz;
UPDATE session_models SET absolute_expires_at = expires_at;
ALTER TABLE session_models ALTER COLUMN absolute_expires_at SET NOT NULL;
ALTER TABLE session_models ADD CONSTRAINT session_absolute_lifetime_check CHECK (expires_at <= absolute_expires_at);

ALTER TABLE session_models DROP CONSTRAINT IF EXISTS session_absolute_lifetime_check;
ALTER TABLE session_models DROP COLUMN IF EXISTS absolute_expires_at;

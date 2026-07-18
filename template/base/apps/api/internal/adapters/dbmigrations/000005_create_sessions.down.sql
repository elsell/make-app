DROP TABLE IF EXISTS session_models;
ALTER TABLE user_models DROP CONSTRAINT IF EXISTS user_models_status_check;
ALTER TABLE user_models DROP COLUMN IF EXISTS status;

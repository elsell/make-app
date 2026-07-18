DROP INDEX IF EXISTS session_models_identity_token_hash_idx;
ALTER TABLE session_models DROP COLUMN IF EXISTS identity_token_hash;

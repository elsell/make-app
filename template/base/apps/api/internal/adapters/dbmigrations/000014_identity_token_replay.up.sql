ALTER TABLE session_models ADD COLUMN identity_token_hash bytea CHECK (identity_token_hash IS NULL OR octet_length(identity_token_hash) = 32);
CREATE UNIQUE INDEX session_models_identity_token_hash_idx ON session_models(identity_token_hash) WHERE identity_token_hash IS NOT NULL;

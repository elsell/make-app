ALTER TABLE user_models ADD COLUMN status text NOT NULL DEFAULT 'active';
ALTER TABLE user_models ADD CONSTRAINT user_models_status_check CHECK (status IN ('active', 'disabled'));

CREATE TABLE session_models (
    token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    user_id text NOT NULL REFERENCES user_models(id) ON DELETE CASCADE,
    scopes text NOT NULL,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL
);
CREATE INDEX session_models_user_active_idx ON session_models(user_id, expires_at) WHERE revoked_at IS NULL;

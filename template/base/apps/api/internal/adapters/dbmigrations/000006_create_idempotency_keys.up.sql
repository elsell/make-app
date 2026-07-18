CREATE TABLE idempotency_models (
    principal_id text NOT NULL REFERENCES user_models(id) ON DELETE CASCADE,
    operation text NOT NULL,
    key text NOT NULL,
    request_hash bytea NOT NULL CHECK (octet_length(request_hash) = 32),
    resource_id text NOT NULL,
    created_at timestamptz NOT NULL,
    PRIMARY KEY (principal_id, operation, key)
);
CREATE INDEX idempotency_models_created_at_idx ON idempotency_models(created_at);

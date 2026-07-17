CREATE TABLE user_models (
    id text PRIMARY KEY,
    email text NOT NULL DEFAULT '',
    display_name text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);
CREATE TABLE identity_models (
    issuer text NOT NULL,
    subject text NOT NULL,
    user_id text NOT NULL UNIQUE REFERENCES user_models(id) ON DELETE CASCADE,
    PRIMARY KEY (issuer, subject)
);
CREATE TABLE resource_models (
    id text PRIMARY KEY,
    domain text NOT NULL DEFAULT 'example',
    owner_user_id text NOT NULL REFERENCES user_models(id) ON DELETE CASCADE,
    name text NOT NULL
);
CREATE INDEX resource_models_domain_idx ON resource_models(domain);
CREATE INDEX resource_models_owner_user_id_idx ON resource_models(owner_user_id);
CREATE TABLE authorization_outbox_models (
    id text PRIMARY KEY,
    resource_type text NOT NULL,
    resource_id text NOT NULL,
    relation text NOT NULL,
    subject_type text NOT NULL,
    subject_id text NOT NULL,
    operation text NOT NULL DEFAULT 'touch',
    attempts integer NOT NULL DEFAULT 0,
    locked_by text,
    locked_until timestamptz,
    completed_at timestamptz,
    created_at timestamptz NOT NULL
);
CREATE INDEX authorization_outbox_pending_idx ON authorization_outbox_models(created_at) WHERE completed_at IS NULL;
CREATE INDEX authorization_outbox_resource_order_idx ON authorization_outbox_models(resource_type, resource_id, created_at) WHERE completed_at IS NULL;

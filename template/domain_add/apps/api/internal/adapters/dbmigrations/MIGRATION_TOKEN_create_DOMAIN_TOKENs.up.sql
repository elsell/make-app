CREATE TABLE __DOMAIN___models (
    id text PRIMARY KEY,
    owner_user_id text NOT NULL REFERENCES user_models(id),
    name text NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);
CREATE INDEX __DOMAIN___models_owner_page_idx ON __DOMAIN___models(owner_user_id, created_at, id);
GRANT SELECT, INSERT, UPDATE, DELETE ON __DOMAIN___models TO app;

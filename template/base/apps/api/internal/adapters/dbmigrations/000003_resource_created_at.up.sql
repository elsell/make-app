ALTER TABLE resource_models ADD COLUMN created_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE resource_models ALTER COLUMN created_at DROP DEFAULT;
CREATE INDEX resource_models_owner_domain_created_id_idx ON resource_models(owner_user_id, domain, created_at, id);

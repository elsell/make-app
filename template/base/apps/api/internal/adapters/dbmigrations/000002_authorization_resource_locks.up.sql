CREATE TABLE authorization_resource_lock_models (
    resource_type text NOT NULL,
    resource_id text NOT NULL,
    PRIMARY KEY (resource_type, resource_id)
);

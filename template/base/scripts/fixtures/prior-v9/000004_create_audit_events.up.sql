CREATE TABLE audit_event_models (
    id text PRIMARY KEY,
    owner_user_id text NOT NULL REFERENCES user_models(id),
    actor_user_id text NOT NULL REFERENCES user_models(id),
    action text NOT NULL CHECK (action IN ('user.provisioned', 'user.deactivated', 'user.profile_synchronized', 'session.created', 'session.revoked', 'resource.listed', 'resource.viewed', 'resource.created', 'resource.updated', 'resource.deleted', 'resource.access_denied', 'authorization.relationship_applied', 'authorization.relationship_failed', 'authorization.dead_letters_listed', 'authorization.dead_letter_requeued')),
    target_type text NOT NULL,
    target_id text NOT NULL,
    outcome text NOT NULL CHECK (outcome IN ('succeeded', 'denied', 'failed')),
    correlation_id text NOT NULL,
    occurred_at timestamptz NOT NULL
);
CREATE INDEX audit_event_models_owner_page_idx ON audit_event_models(owner_user_id, occurred_at, id);
CREATE INDEX audit_event_models_actor_page_idx ON audit_event_models(actor_user_id, occurred_at, id);
CREATE FUNCTION reject_audit_event_mutation() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'audit events are append-only'; END; $$ LANGUAGE plpgsql;
CREATE TRIGGER audit_events_append_only BEFORE UPDATE OR DELETE ON audit_event_models FOR EACH ROW EXECUTE FUNCTION reject_audit_event_mutation();
CREATE TRIGGER audit_events_no_truncate BEFORE TRUNCATE ON audit_event_models FOR EACH STATEMENT EXECUTE FUNCTION reject_audit_event_mutation();
GRANT USAGE ON SCHEMA public TO app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO app;
REVOKE UPDATE, DELETE, TRUNCATE ON audit_event_models FROM app;
REVOKE INSERT, UPDATE, DELETE, TRUNCATE ON schema_migrations FROM app;

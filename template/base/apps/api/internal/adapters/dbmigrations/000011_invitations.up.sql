ALTER TABLE user_models ADD COLUMN invitation_admin boolean NOT NULL DEFAULT false;

ALTER TABLE audit_event_models DROP CONSTRAINT audit_event_models_action_check;
ALTER TABLE audit_event_models ADD CONSTRAINT audit_event_models_action_check CHECK (action IN ('user.provisioned', 'user.deactivated', 'user.profile_synchronized', 'session.created', 'session.revoked', 'resource.listed', 'resource.viewed', 'resource.created', 'resource.updated', 'resource.deleted', 'resource.access_denied', 'authorization.relationship_applied', 'authorization.relationship_failed', 'authorization.dead_letters_listed', 'authorization.dead_letter_requeued', 'invitation.created', 'invitation.listed', 'invitation.revoked', 'invitation.consumed'));

CREATE TABLE invitation_models (
    id text PRIMARY KEY,
    email text NOT NULL,
    created_by_user_id text NOT NULL REFERENCES user_models(id),
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    consumed_by_user_id text REFERENCES user_models(id),
    consumed_at timestamptz,
    revoked_by_user_id text REFERENCES user_models(id),
    revoked_at timestamptz,
    CHECK (expires_at > created_at),
    CHECK ((consumed_by_user_id IS NULL) = (consumed_at IS NULL)),
    CHECK ((revoked_by_user_id IS NULL) = (revoked_at IS NULL)),
    CHECK (consumed_at IS NULL OR revoked_at IS NULL)
);
CREATE UNIQUE INDEX invitation_active_email_idx ON invitation_models(email) WHERE consumed_at IS NULL AND revoked_at IS NULL;
CREATE INDEX invitation_admin_page_idx ON invitation_models(created_by_user_id, created_at, id);
GRANT SELECT, INSERT, UPDATE ON invitation_models TO app;

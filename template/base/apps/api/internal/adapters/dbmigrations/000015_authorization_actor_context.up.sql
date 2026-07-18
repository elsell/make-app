ALTER TABLE authorization_outbox_models ADD COLUMN owner_user_id text;
ALTER TABLE authorization_outbox_models ADD COLUMN actor_user_id text;
UPDATE authorization_outbox_models SET owner_user_id=subject_id, actor_user_id=subject_id;
ALTER TABLE authorization_outbox_models ALTER COLUMN owner_user_id SET NOT NULL;
ALTER TABLE authorization_outbox_models ALTER COLUMN actor_user_id SET NOT NULL;

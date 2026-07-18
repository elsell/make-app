ALTER TABLE authorization_outbox_models DROP COLUMN IF EXISTS actor_user_id;
ALTER TABLE authorization_outbox_models DROP COLUMN IF EXISTS owner_user_id;

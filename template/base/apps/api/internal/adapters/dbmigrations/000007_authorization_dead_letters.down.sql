DROP INDEX IF EXISTS authorization_outbox_dead_letter_idx;
DROP INDEX IF EXISTS authorization_outbox_pending_idx;
CREATE INDEX authorization_outbox_pending_idx ON authorization_outbox_models(created_at) WHERE completed_at IS NULL;
ALTER TABLE authorization_outbox_models DROP COLUMN IF EXISTS failure_code;
ALTER TABLE authorization_outbox_models DROP COLUMN IF EXISTS dead_lettered_at;

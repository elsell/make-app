ALTER TABLE authorization_outbox_models ADD COLUMN dead_lettered_at timestamptz;
ALTER TABLE authorization_outbox_models ADD COLUMN failure_code text NOT NULL DEFAULT '';
DROP INDEX authorization_outbox_pending_idx;
CREATE INDEX authorization_outbox_pending_idx ON authorization_outbox_models(created_at) WHERE completed_at IS NULL AND dead_lettered_at IS NULL;
CREATE INDEX authorization_outbox_dead_letter_idx ON authorization_outbox_models(dead_lettered_at) WHERE dead_lettered_at IS NOT NULL;

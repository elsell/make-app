DROP INDEX IF EXISTS authorization_outbox_resource_created_idx;
DROP TRIGGER IF EXISTS authorization_outbox_assign_order ON authorization_outbox_models;
DROP FUNCTION IF EXISTS assign_authorization_outbox_order();

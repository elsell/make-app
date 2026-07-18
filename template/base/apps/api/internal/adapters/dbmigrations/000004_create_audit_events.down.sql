DROP TRIGGER IF EXISTS audit_events_no_truncate ON audit_event_models;
DROP TRIGGER IF EXISTS audit_events_append_only ON audit_event_models;
DROP FUNCTION IF EXISTS reject_audit_event_mutation();
DROP TABLE IF EXISTS audit_event_models;

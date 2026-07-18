DROP TRIGGER IF EXISTS audit_retention_summaries_no_truncate ON audit_retention_run_models;
DROP TRIGGER IF EXISTS audit_retention_summaries_append_only ON audit_retention_run_models;
DROP TRIGGER IF EXISTS audit_retention_perform_run ON audit_retention_run_models;
DROP FUNCTION IF EXISTS reject_audit_retention_summary_mutation();
DROP FUNCTION IF EXISTS perform_audit_retention_run();
DROP TABLE IF EXISTS audit_retention_run_models;
REVOKE ALL ON audit_event_models FROM app_audit_retention;
CREATE OR REPLACE FUNCTION reject_audit_event_mutation() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'audit events are append-only';
END;
$$ LANGUAGE plpgsql;

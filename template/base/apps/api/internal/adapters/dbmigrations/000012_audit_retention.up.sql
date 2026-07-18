CREATE TABLE audit_retention_run_models (
    id text PRIMARY KEY,
    cutoff timestamptz NOT NULL,
    batch_limit integer NOT NULL CHECK (batch_limit BETWEEN 1 AND 10000),
    dry_run boolean NOT NULL,
    deleted_count integer NOT NULL CHECK (deleted_count >= 0),
    completed_at timestamptz NOT NULL
);

CREATE OR REPLACE FUNCTION reject_audit_event_mutation() RETURNS trigger AS $$
BEGIN
    IF session_user = 'app_audit_retention' AND current_user <> session_user AND TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RAISE EXCEPTION 'audit events are append-only';
END;
$$ LANGUAGE plpgsql;

CREATE FUNCTION perform_audit_retention_run() RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, public
AS $$
BEGIN
    IF NEW.cutoff > clock_timestamp() - interval '30 days' THEN
        RAISE EXCEPTION 'audit retention cutoff must be at least 30 days old';
    END IF;

    NEW.completed_at := clock_timestamp();
    IF NEW.dry_run THEN
        SELECT count(*) INTO NEW.deleted_count
        FROM public.audit_event_models
        WHERE occurred_at < NEW.cutoff;
        RETURN NEW;
    END IF;

    WITH candidates AS (
        SELECT id
        FROM public.audit_event_models
        WHERE occurred_at < NEW.cutoff
        ORDER BY occurred_at ASC, id ASC
        LIMIT NEW.batch_limit
        FOR UPDATE SKIP LOCKED
    ), deleted AS (
        DELETE FROM public.audit_event_models AS events
        USING candidates
        WHERE events.id = candidates.id
        RETURNING events.id
    )
    SELECT count(*) INTO NEW.deleted_count FROM deleted;
    RETURN NEW;
END;
$$;

REVOKE ALL ON FUNCTION perform_audit_retention_run() FROM PUBLIC;

CREATE TRIGGER audit_retention_perform_run
BEFORE INSERT ON audit_retention_run_models
FOR EACH ROW EXECUTE FUNCTION perform_audit_retention_run();

CREATE FUNCTION reject_audit_retention_summary_mutation() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'audit retention summaries are append-only';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_retention_summaries_append_only
BEFORE UPDATE OR DELETE ON audit_retention_run_models
FOR EACH ROW EXECUTE FUNCTION reject_audit_retention_summary_mutation();

CREATE TRIGGER audit_retention_summaries_no_truncate
BEFORE TRUNCATE ON audit_retention_run_models
FOR EACH STATEMENT EXECUTE FUNCTION reject_audit_retention_summary_mutation();

GRANT USAGE ON SCHEMA public TO app_audit_retention;
GRANT SELECT, INSERT ON audit_retention_run_models TO app_audit_retention;
REVOKE ALL ON audit_event_models FROM app_audit_retention;
REVOKE ALL ON audit_retention_run_models FROM app;

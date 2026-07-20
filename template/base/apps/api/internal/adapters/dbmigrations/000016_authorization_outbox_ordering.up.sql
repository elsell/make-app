CREATE FUNCTION assign_authorization_outbox_order() RETURNS trigger
LANGUAGE plpgsql
SECURITY INVOKER
SET search_path = pg_catalog, public
AS $$
DECLARE
    previous_created_at timestamptz;
BEGIN
    INSERT INTO public.authorization_resource_lock_models(resource_type, resource_id)
    VALUES (NEW.resource_type, NEW.resource_id)
    ON CONFLICT DO NOTHING;

    PERFORM 1
    FROM public.authorization_resource_lock_models
    WHERE resource_type = NEW.resource_type AND resource_id = NEW.resource_id
    FOR UPDATE;

    SELECT max(created_at) INTO previous_created_at
    FROM public.authorization_outbox_models
    WHERE resource_type = NEW.resource_type AND resource_id = NEW.resource_id;

    NEW.created_at := GREATEST(
        clock_timestamp(),
        COALESCE(previous_created_at + INTERVAL '1 microsecond', '-infinity'::timestamptz)
    );
    RETURN NEW;
END;
$$;

CREATE TRIGGER authorization_outbox_assign_order
BEFORE INSERT ON authorization_outbox_models
FOR EACH ROW EXECUTE FUNCTION assign_authorization_outbox_order();

CREATE INDEX authorization_outbox_resource_created_idx
ON authorization_outbox_models(resource_type, resource_id, created_at DESC);

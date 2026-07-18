# __DOMAIN__ Persistence and Transport Specification

`__DOMAIN__` owns the `__DOMAIN___models` PostgreSQL table and its repository.
No operation uses the example domain's generic `resource_models` storage.
Repository reads require an owner user ID and use stable `(created_at, id)`
keyset pagination. Creates atomically reserve the authenticated principal's
idempotency key and canonical request digest, insert the domain row, enqueue the
SpiceDB owner relationship, and append the typed audit event. Same-digest retries
return the original entity; key reuse with another digest conflicts. The
idempotency reservation and authorization outbox row carry explicit non-null
creation timestamps required by the shared schema. Updates commit their typed
audit event atomically. Deletes atomically remove the domain row, enqueue the
SpiceDB relationship deletion, and append their typed audit event so authorization
cleanup cannot be lost.

The generated Huma registrar, DTOs, and mapper are compile-safe extension points.
They are not registered automatically. Implement the `routes.Service` application
port with domain behavior, authentication, authorization coordination, and audit
construction, then explicitly register it in API composition. Sharing is not
generated and must be specified before it is added.
Application services use `internal/app/shared` for correlation-aware audit event
construction instead of duplicating request-context or clock mechanics.

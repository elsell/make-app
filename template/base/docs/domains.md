# Complete a generated domain

Start with a product spec, then generate the mechanical slice:

```sh
make-app domain add habit --fields 'title:string,active:bool,target:int,due_at:time'
```

The command creates typed domain fields, a dedicated table migration, a GORM
repository, application ports, Huma DTOs/routes, mappers, and focused tests. It
does not invent authorization or sharing behavior.

Complete the slice in this order:

1. Specify commands, invariants, ownership, sharing, audit actions, and errors.
2. Implement the application service in `apps/api/internal/app/habit` using the
   generated repository port, authentication port, SpiceDB authorization port,
   audit helpers, injected clock, and idempotency boundary.
3. Wire its repository and service in `apps/api/cmd/server`, then call the
   generated route registrar from HTTP API composition.
4. Extend the single embedded SpiceDB schema at
   `apps/api/internal/adapters/spicedb/schema.zed` with the domain's explicit
   relations and permissions. The schema job and readiness check both consume
   this artifact. Queue `AuthorizationChange` values with the initiating owner
   and actor plus the typed relation/subject; the generic outbox forwards owner,
   viewer, editor, and other schema-defined relationships without rewriting its
   retry/dead-letter machinery. Never grant access merely because a row exists.
5. Run `make migrate`, `make generate`, and add a client adapter over the
   generated API package.
6. Add adversarial endpoint tests for unauthenticated, malformed, cross-user,
   insufficient-permission, dependency-failure, and legitimate calls.
7. Run `make verify` and `make acceptance`.

Use `make-app example remove` after the real product slice replaces the
demonstration API. The command adds a forward migration; it does not rewrite
already-applied migration history.

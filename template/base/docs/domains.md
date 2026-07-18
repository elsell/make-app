# Complete a generated domain

Start with a product spec, then generate the mechanical slice:

```sh
make-app domain add habit --fields 'title:string,active:bool,target:int,due_at:time'
```

The command creates typed domain fields, a dedicated table migration, a GORM
repository, application ports and dependency bundle, Huma DTOs/routes, mappers,
composition wiring, and focused tests. Runtime and OpenAPI composition register
the routes immediately. The initial service authenticates callers and then
returns `503` because no product authorization policy exists yet; it never grants
access just because the repository is wired.

Complete the slice in this order:

1. Specify commands, invariants, ownership, sharing, audit actions, and errors.
2. Replace the fail-closed methods in `apps/api/internal/app/habit/service.go`
   with the specified application behavior. Its generated `Dependencies` value
	already injects the repository, authentication, SpiceDB authorization, audit,
	clock, probe, ID, authorization outbox, per-resource serializer,
	authorization-worker, and cursor-signing capabilities.
3. Keep `apps/api/internal/generated/domains.go` generator-owned. Repository
   construction and Huma registration are regenerated from `.make-app.json`;
   application policy belongs in the domain service, not composition code.
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
   Preserve the generated missing/malformed-session and unconfigured-policy
   cases when expanding that suite.
7. Run `make verify` and `make acceptance`.

Use `make-app example remove` after the real product slice replaces the
demonstration API. The command adds a forward migration; it does not rewrite
already-applied migration history.

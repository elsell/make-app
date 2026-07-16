# Make App Generator Specification

## Purpose

`make-app` bootstraps production-shaped application monorepos with a Go API,
PostgreSQL, SpiceDB, OIDC, generated TypeScript API contracts, a web client, and
an Expo native client. Generated applications own their source code and do not
depend on a Make App runtime framework.

## Commands

- `make-app new NAME --module MODULE [--output DIR]` creates a new repository.
- `make-app domain add NAME [--dir DIR]` adds a user-owned domain resource.
- Generation refuses to overwrite non-empty destinations or existing domains.

## Generated baseline

- OIDC identities are keyed by issuer and subject and provision a local user.
- `GET /v1/me` returns the authenticated local user.
- PostgreSQL stores users and application resources.
- SpiceDB is the authorization decision point for resource access.
- A generated `example` resource demonstrates owner-only create, read, and list.
- Huma produces OpenAPI; pinned `openapi-typescript` produces client types.
- Web and Expo clients use the generated API package.
- Docker Compose starts PostgreSQL, SpiceDB, Dex, API, and web services.
- Lefthook and GitHub Actions run formatting, tests, generation drift checks,
  TypeScript checks, and builds.

## Security guarantees

- Tokens validate signature, issuer, audience, and expiry through OIDC discovery.
- Routes do not trust identity headers supplied by callers.
- Resource reads require a SpiceDB permission check and hide inaccessible IDs.
- Repository list operations require the authenticated owner ID.
- Local development uses Dex; production settings come from environment variables.


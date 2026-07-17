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
- Generated authenticator tests cryptographically exercise valid tokens plus
  invalid signatures, wrong issuers, wrong audiences, expiry, and empty subjects.
- Routes do not trust identity headers supplied by callers.
- Resource reads require a SpiceDB permission check and hide inaccessible IDs.
- Repository list operations require the authenticated owner ID.
- Local development uses Dex; production settings come from environment variables.

## Definition of ready

A release is not ready unless an empty-directory generation test proves all of
the following without manual source edits:

- generator unit tests and deterministic snapshot/structure checks pass;
- generated Go formatting, static analysis, unit tests, race tests, and builds pass;
- generated TypeScript checks, tests, OpenAPI drift checks, and production builds pass;
- pinned Compose configuration starts healthy PostgreSQL, SpiceDB, Dex, API, and web services;
- a real OIDC authorization-code-with-PKCE flow provisions exactly one local user;
- valid access, missing token, malformed token, invalid signature, wrong issuer,
  wrong audience, expired token, and concurrent first-login behavior are tested;
- owner creation, list, detail, update, and delete work through the public API;
- a second user cannot discover, read, change, delete, or forge ownership of the
  first user's resource, and denials do not reveal resource existence;
- SpiceDB unavailability fails closed, and interrupted relationship writes recover
  from a durable transactional outbox without orphaning or granting resources;
- PostgreSQL data and SpiceDB relationships survive a full generated-stack restart,
  including reauthentication after the local provider restarts;
- migrations apply to an empty database and upgrade from the prior released schema;
- generated hooks are installed and CI runs the same authoritative verification command;
- dependencies, actions, tools, and runtime images are immutable and lockfiles are generated;
- every third-party CI action is selected by a reviewed immutable commit SHA, and
  CI installs JavaScript dependencies from the generated frozen lockfile;
- the installed pre-commit hook and CI invoke the same `make verify` release gate;
- the generator and every generated repository fail closed when an npm package
  or Go module in the resolved graph is less than fourteen days old;
- an age exception requires an exact ecosystem/name/version entry with a reason
  and compensating verification in `dependency-age-allowlist.json` plus a
  corresponding specification update;
- web and mobile clients complete sign-in, refresh, `/v1/me`, authorized resource
  access, expiry handling, and sign-out through their adapters.

The live acceptance harness must run on every generator release. A skipped boundary
test is a release failure unless a reviewed specification records the temporary
exception, owner, risk, and removal date.

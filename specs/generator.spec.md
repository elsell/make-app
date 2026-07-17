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
- PostgreSQL schema changes are reviewed `golang-migrate` files applied by a
  one-shot migration service; API startup has no schema mutation privileges.
- A generated `example` resource demonstrates owner-only create, read, and list.
- Resource lists use stable opaque cursor pagination with a bounded page size,
  deterministic ordering, continuation metadata, and invalid-cursor rejection.
- Huma produces OpenAPI; pinned `openapi-typescript` produces client types.
- Huma's interactive documentation is configured with a dedicated public OIDC
  client and authorization-code PKCE so protected routes are usable from docs;
  authorization and token endpoints come from OIDC discovery rather than
  issuer-path assumptions.
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
- pagination proves default and maximum limits, complete gap-free traversal,
  stable continuation under inserts, and rejection of malformed cursors;
- a second user cannot discover, read, change, delete, or forge ownership of the
  first user's resource, and denials do not reveal resource existence;
- SpiceDB unavailability fails closed, and interrupted relationship writes recover
  from a durable transactional outbox without orphaning or granting resources;
- concurrent outbox workers use owned expiring leases, cannot complete one
  another's claims, and recover an abandoned claim after its lease expires;
- every external authorization write is serialized per resource by a durable
  PostgreSQL lock held through the SpiceDB call, so an expired or delayed worker
  cannot reorder TOUCH and DELETE operations or resurrect access;
- PostgreSQL data and SpiceDB relationships survive a full generated-stack restart,
  including reauthentication after the local provider restarts;
- migrations apply to an empty database and upgrade from the prior released schema;
- generated hooks are installed and CI runs the same authoritative verification command;
- dependencies, actions, tools, and runtime images are immutable and lockfiles are generated;
- Go-based release tools live in a dedicated checked-in module so their complete
  transitive graph is pinned, age-gated, and reviewed like application dependencies;
- every third-party CI action is selected by a reviewed immutable commit SHA, and
  CI installs JavaScript dependencies from the generated frozen lockfile;
- the installed pre-commit hook and CI invoke the same `make verify` release gate;
- the generator and every generated repository fail closed when an npm package
  or Go module in the resolved graph is less than fourteen days old;
- an age exception requires an exact ecosystem/name/version entry with a reason
  and compensating verification in `dependency-age-allowlist.json` plus a
  corresponding specification update;
- dependency metadata requests use bounded connection and total timeouts so the
  supply-chain gate fails closed instead of hanging a hook or CI job;
- web and mobile clients complete sign-in, refresh, `/v1/me`, authorized resource
  access, expiry handling, and sign-out through their adapters.
- interactive API docs complete OIDC PKCE and successfully invoke `/v1/me` plus
  a protected resource operation without a client secret.
- the documentation token relay accepts the pinned Scalar public-client request
  shape, discards caller-supplied client and redirect identities, and constructs
  only a fixed authorization-code PKCE exchange for the configured docs client.

The live acceptance harness must run on every generator release. A skipped boundary
test is a release failure unless a reviewed specification records the temporary
exception, owner, risk, and removal date.
Generated CI also runs the live Compose acceptance harness, including an actual
pinned Scalar browser session that clicks Authorize, completes Dex login, and
uses Try It for `/v1/me` and a protected resource list. Playwright is an exact,
age-gated development dependency; that reviewed package fixes the downloaded
Chromium revision rather than resolving a floating browser release.

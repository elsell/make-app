# Make App Generator Specification

## Purpose

`make-app` bootstraps production-shaped application monorepos with a Go API,
PostgreSQL, SpiceDB, OIDC, generated TypeScript API contracts, a web client, and
an Expo native client. Generated applications own their source code and do not
depend on a Make App runtime framework.

## Commands

- `make-app new NAME --module MODULE [--output DIR]` creates a new repository.
- `make-app domain add NAME [--dir DIR]` adds a user-owned vertical-slice
  scaffold. It generates a typed entity, application repository and HTTP service
  ports, domain-owned GORM repository/model, dedicated PostgreSQL table migration,
  DTOs, mappers, Huma route registrar, and focused tests. It never registers the
  new domain against the example domain's shared storage or generic routes. The
  developer writes the application service and explicitly wires the registrar
  after specifying behavior and sharing.
- Added-domain migrations use the next unused monotonic migration version and
  never overwrite or reuse an existing version.
- Domain addition is failure-atomic: generated files are formatted in staging,
  and any failed installation or metadata update removes staged additions and
  restores modified generator metadata so the same command can be retried.
- Generation refuses to overwrite non-empty destinations or existing domains.

## Generated baseline

- OIDC identities are keyed by issuer and subject and provision a local user.
- Interactive clients use authorization code with PKCE and an application session
  boundary; OIDC identity tokens are not long-lived API bearer credentials.
  Machine credentials are separate, audience-bound, and least privilege.
- Application-session rotation preserves a configurable, non-extendable family
  deadline of no more than seven days, after which OIDC authentication is required.
- Generated web and mobile clients detect a family-deadline-capped refresh, stop
  rotating that family, and expire it locally at the returned deadline. Short
  session lifetimes use bounded midpoint scheduling instead of a one-second loop.
- Create commands require a principal-and-operation-scoped idempotency key.
  Exact replays return the original result and conflicting reuse is rejected.
- Generated domain deletes atomically remove their own row, enqueue the matching
  SpiceDB relationship deletion, and append their audit event.
- Identity profiles have specified claim synchronization, invitation linkage,
  configurable disable/delete lifecycle, and audit behavior.
- Existing-only provisioning is usable on a fresh deployment through a validated,
  normalized environment-backed invited-email allowlist; it never becomes open
  registration when the allowlist is empty.
- `GET /v1/me` returns the authenticated local user.
- PostgreSQL stores users and application resources.
- An append-only audit stream is a mandatory platform primitive. User
  provisioning and every authenticated resource read, list, create, update,
  delete, and denied authorization decision produce a structured audit event.
  Business mutations and their successful audit events commit atomically.
- SpiceDB is the authorization decision point for resource access.
- Local and self-hosted topology places a typed credential-checking gRPC proxy
  before SpiceDB so the API runtime credential cannot write authorization schema;
  the upstream administration credential is absent from the API process.
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
- Interactive documentation can be disabled through validated configuration.
- Web and Expo clients use the generated API package.
- Internationalization is a non-optional presentation-layer invariant. A shared,
  typed locale package supplies web and Expo copy, locale negotiation, fallback,
  interpolation, pluralization, and locale-aware number/date formatting. The
  generated baseline includes complete English and Spanish catalogs.
- User-facing client copy must come from locale catalogs. Generated structural
  checks reject literal JSX/Svelte copy and inconsistent or incomplete catalogs;
  every API, health, routing, CORS, and OIDC relay error exposes a stable
  machine-readable code rather than relying on localized strings.
- Docker Compose starts PostgreSQL, SpiceDB, Dex, API, and web services. The web
  service runs package installation in non-interactive CI mode so a host-mounted
  workspace with incompatible modules is repaired without a terminal prompt. It
  runs as a configurable unprivileged host UID/GID so generated dependency and
  build artifacts remain removable by the developer. Bootstrap and Compose use
  the same repository-local pnpm store so downloaded content persists with the
  generated workspace. Live acceptance permits a bounded ten-minute cold-start
  window for a slow package registry while still failing closed on readiness.
  Generated Make and acceptance entrypoints derive the web container UID/GID
  from the invoking host user rather than relying on the Compose fallback, so
  non-1000 CI runners and developer accounts retain write access. The container
  supplies writable temporary HOME, XDG cache, and Corepack directories because
  arbitrary numeric UIDs need not have an image passwd entry. Acceptance executes
  the exact pinned image as UID/GID 1001 to prove the runtime contract.
  Browser acceptance invokes its checked-in Node entrypoint directly after the
  pinned Playwright install so it cannot trigger a competing pnpm modules purge
  while the web container is serving from that workspace.
- Lefthook and GitHub Actions run formatting, tests, generation drift checks,
  TypeScript checks, and builds.
- Rate limits, PostgreSQL pool sizes and lifetimes, account lifecycle, and
  observability exporters are environment-backed and validated.
- Liveness is process-only; readiness verifies migrations and required security
  dependencies. Structured access logs, metrics, traces, correlation IDs, and
  typed domain probes are injectable and fan-out capable. Optional validated
  OTLP/HTTP exporters emit real W3C-context traces and metrics.
- Authorization outbox retries have a configurable maximum-attempt dead-letter
  policy, cursor-paginated affected-owner visibility, atomically preclaimed
  audited replay, and metrics.
- The web app has a pinned non-root production image. Generated CI builds and
  scans API and web images and includes an immutable-action release workflow.
- SvelteKit owns nonce-based Content Security Policy generation. The server hook
  may adjust only `connect-src` for validated runtime API and OIDC origins so the
  production app hydrates without allowing inline script execution.
- Installation docs pin a concrete make-app release and never use `@latest`.

## Security guarantees

- Tokens validate signature, issuer, audience, and expiry through OIDC discovery.
- Generated authenticator tests cryptographically exercise valid tokens plus
  invalid signatures, wrong issuers, wrong audiences, expiry, and empty subjects.
- Routes do not trust identity headers supplied by callers.
- Resource reads require a SpiceDB permission check and hide inaccessible IDs.
- Audit history is scoped to events performed by the authenticated user or
  events affecting resources they own; callers cannot enumerate another user's
  activity through the audit API.
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
- successful reads and writes, user provisioning, and denied cross-user
  attempts appear in immutable, cursor-paginated audit history without leaking
  another user's unrelated activity;
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
- browser acceptance drives the generated web client itself through Dex PKCE,
  callback exchange, application-session creation, and authenticated rendering;
  the local provider explicitly permits that public client's web origin.
- live acceptance rotates an application session, proves the old credential is
  immediately rejected, and continues with the replacement credential.
- PostgreSQL acceptance proves a rotation cannot extend a session beyond its
  original absolute family deadline.
- web and mobile select a supported device/browser locale, fall back safely to
  English, render translated copy and interpolation/plurals, and pass the
  untranslated-copy and catalog-parity structural gate;
- interactive API docs complete OIDC PKCE and successfully invoke `/v1/me` plus
  a protected resource operation without a client secret.
- the documentation token relay accepts the pinned Scalar public-client request
  shape, discards caller-supplied client and redirect identities, and constructs
  only a fixed authorization-code PKCE exchange for the configured docs client.

The live acceptance harness must run on every generator release. A skipped boundary
test is a release failure unless a reviewed specification records the temporary
exception, owner, risk, and removal date.
The PostgreSQL acceptance invocation runs the complete persistence-adapter test
package with a real DSN; name-based filtering must not silently omit a new
security boundary test.
Generator acceptance adds a real domain before bootstrap and runs every generated
domain repository's PostgreSQL integration test, including atomic create/delete
outbox, idempotency, timestamp, and audit guarantees.
The migration proof explicitly applies the prior release's complete migration
set and ledger state before invoking the current migrator; a hand-built baseline
that skips intervening migrations is not an upgrade test.
Generated CI also runs the live Compose acceptance harness, including an actual
pinned Scalar browser session that clicks Authorize, completes Dex login, and
uses Try It for `/v1/me` and a protected resource list. Playwright is an exact,
age-gated development dependency; that reviewed package fixes the downloaded
Chromium revision rather than resolving a floating browser release.
The same browser acceptance opens the generated web client with a regional
Spanish browser locale and proves base-locale negotiation, the document language,
and translated UI copy at the real rendering boundary.
Generated release planning reruns that live acceptance gate before publication.
The publish job builds and scans both container images for high and critical
vulnerabilities with a full-SHA-pinned scanner before pushing immutable images.
The live harness starts and waits for the generated web service before these
checks; API-only readiness is not sufficient for frontend acceptance.
Each live acceptance invocation uses a unique Compose project name and removes
its project before startup and on exit, so named volumes and containers from an
interrupted run cannot contaminate migration or persistence proof. Because the
browser-visible local OIDC issuer intentionally uses fixed localhost ports,
acceptance also holds a bounded host lock; simultaneous invocations serialize
instead of colliding or connecting to another run's services.
The generated harness stops the API while direct PostgreSQL adapter integration
tests run, preventing the live authorization worker from racing those tests for
outbox rows, then restarts the API and proves readiness before HTTP acceptance.
Generated behavioral tests cover fallback, interpolation, plural forms,
locale-aware numbers/dates, and the Expo device-locale adapter; locale-dependent
SSR responses are required to emit `Vary: Accept-Language`.

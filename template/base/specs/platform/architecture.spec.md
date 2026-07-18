# Platform Architecture Specification

## Structure

The API uses hexagonal architecture. Domain packages contain business concepts;
application packages coordinate use cases; ports describe required capabilities;
adapters implement transport, persistence, identity, authorization, and telemetry;
bootstrap composes adapters and owns lifecycle.

Added domains are composed through generated dependency injection rather than
package globals. Their application service receives authentication,
authorization, persistence, audit, clock, observability, ID generation,
authorization outbox, per-resource serializer, authorization-worker, and
cursor-signing capabilities through a typed `Dependencies` value. The generated registry constructs the dedicated
repository and registers the domain route adapter for runtime and OpenAPI use.

Generation does not invent a domain authorization policy. Until the developer
implements that policy, every generated operation authenticates the application
session and then returns a typed policy-not-configured error. This placeholder is
fail-closed: unauthenticated and malformed credentials return 401, while a valid
principal receives 503 and no repository or SpiceDB operation runs.
Only `ErrInvalidCredential` is translated to 401. Authentication dependency and
context errors retain their original classification and cannot trick clients
into discarding a legitimate session.

## Baseline stack

The checked-in local cursor-signing and metrics-token sentinel values are valid
only when database, OIDC, and SpiceDB insecure modes are all explicitly enabled.
Any secure deployment fails startup until both known values are replaced.

`apps/api/internal/adapters/spicedb/schema.zed` is the sole authorization-schema
artifact and is embedded by the SpiceDB adapter for both schema application and
readiness comparison. Authorization outbox changes carry relation, subject,
resource owner, and initiating actor independently so non-owner sharing never
misattributes audit history. Outbox completion validates audit ownership and
attribution against those explicit owner and actor fields, never against the
relationship subject.

- Go API with Huma-generated OpenAPI.
- PostgreSQL with GORM and versioned migrations.
- OIDC authentication with immutable issuer/subject identity mapping.
- SpiceDB authorization behind a project-owned port.
- TypeScript contracts generated with pinned `openapi-typescript` and consumed
  using pinned `openapi-fetch`.
  The shared authenticated fetch adapter preserves every header serialized by
  that generated contract and merges the current application-session bearer
credential without discarding idempotency or future domain headers.

Shared request-rate state has a hard configured principal bound. Expired state
may be removed, but capacity pressure never evicts an active principal: a new
unknown source fails closed until capacity becomes stale. Source identity uses
the direct peer by default. Forwarded client addresses are honored only when
the immediate peer and every skipped proxy hop match explicitly configured
trusted proxy CIDRs; untrusted or malformed forwarding headers are ignored.
- Separate SvelteKit web and Expo React Native applications.
- A shared typed internationalization package consumed by both clients. English
  is the safe fallback and the generated baseline also contains a complete
Spanish catalog. Browser/device locale negotiation selects only explicitly
supported locales; unsupported and malformed locale values fall back safely.
Server-rendered web responses honor `Accept-Language` quality values, exclude
zero-quality languages, and emit `Vary: Accept-Language` so caches cannot mix
localized representations.

Internationalization is a presentation-layer invariant. All client-visible copy,
including errors and accessibility labels, comes from locale catalogs. Catalogs
must have identical keys, interpolation parameters, and plural forms. The shared
adapter owns interpolation and locale-aware plural, number, date, and time rules.
API and domain layers expose stable error codes and structured values rather than
pre-localized presentation sentences.
Every HTTP failure boundary includes a stable `code`: Huma operation and
validation problems use the project-owned RFC 9457 model; health, routing, and
CORS failures use the same model; and OIDC relay failures retain OAuth-compatible
fields while adding the stable code.

Audit is a first-class application port and append-only persistence model, not
HTTP access logging. User provisioning and every authenticated domain list,
detail read, command, and denied authorization decision emit a structured event
with immutable ID, owner, actor, action, target, outcome, correlation ID, and
UTC occurrence time. A successful business mutation and its audit event commit
in one PostgreSQL transaction. If the audit write fails, the mutation fails.
Read and denial events must be durably appended before their response is
returned. Health checks, documentation assets, and unauthenticated malformed
traffic are excluded because they do not have a trustworthy application actor.
Audit-history listing is also excluded because recursively auditing observation
would mutate the stream and prevent stable traversal.

Audit history uses the same signed, principal-bound keyset pagination guarantees
as other collections. A user may see events they performed and events affecting
resources they own. Audit records have no update or delete application port, and
PostgreSQL rejects direct row updates and deletes. Retention or export must be a
separately specified privileged lifecycle rather than ordinary CRUD.
PostgreSQL also rejects audit truncation. Runtime and migration database roles
are distinct; the runtime role may select and insert audit records but cannot
update, delete, truncate, change schema, or mutate the migration ledger.
Authenticated operations that generate audit writes pass through a bounded
per-principal limiter, including identity exchange/profile synchronization and
session revocation. The PostgreSQL-coordinated enforcement tier preserves the
configured limit across replicas; operators also configure audit-write-rate and
database-capacity alerts.
Every `/v1` interaction also passes through a separately configurable bounded
source limiter before transport decoding. Health checks are not counted against
the application budget. The default adapter coordinates fixed windows through
PostgreSQL so adding API replicas cannot multiply either limit. An in-process
implementation is retained only as an injected test or explicitly selected
single-process development adapter.

All runtime configuration is environment-backed and validated at startup.
Infrastructure dependencies are replaceable adapters.
Shared PostgreSQL rate-limit windows map the database's complete `(scope,
principal_hash)` identity and every update must affect exactly one row. An
incomplete ORM identity must fail closed without becoming an unscoped update or
an accidental global denial.
Observability uses a typed injected probe port with fan-out to structured JSON
logging, a bounded authenticated Prometheus registry, and optional environment-
configured OTLP/HTTP trace and metric exporters. The OpenTelemetry adapter uses
real W3C trace context and records domain probes as span events and metrics;
exporter shutdown flushes within a bounded deadline. Access events include method, route,
status, duration, correlation ID, and trace ID, but never credentials, query
values, or bodies. The metrics endpoint uses a dedicated bearer credential.
Readiness executes bounded checks through injected PostgreSQL and SpiceDB health
ports and returns unavailable whenever either security dependency cannot be reached,
the database migration ledger is dirty or behind the generated migration set, or
SpiceDB does not contain the exact generated authorization schema.
Collection APIs use versioned, HMAC-authenticated keyset cursors rather than
offsets. A cursor is bound to its principal and domain and carries a first-page
snapshot boundary plus the last immutable `(created_at, id)` key. Pages default
to 50 entries, accept at most 100, fetch one extra row to determine continuation,
and exclude inserts after traversal began. Malformed, forged, cross-principal,
cross-domain, and out-of-range cursors are client errors.
List envelopes always encode `data` as a JSON array, including `[]` for an empty
page; generated clients never receive `null` for a collection contract.
Successful REST resource and invitation creation returns `201 Created` and the
created representation. Credential exchange and command-style POST operations
retain their protocol-appropriate status instead of being treated as resources.
The separately deployed web image reads its API and OIDC public settings from
runtime environment variables for every authentication and API adapter; it does
not bake one API endpoint into the production bundle.
Its production Content Security Policy preserves SvelteKit-managed nonces for
framework bootstrap scripts while restricting all default sources. Runtime API
and OIDC origins may extend only `connect-src`; handwritten CSP headers must not
disable hydration or require `unsafe-inline` script execution.
The public HTTP server sets bounded header, request, response, idle, and shutdown
timeouts plus a bounded maximum header size so slow or oversized clients cannot
hold resources indefinitely.
API responses disable MIME sniffing and sensitive response caching. Request
bodies are capped before transport decoding, and rejected oversized payloads do
not reach application services.

Database schema changes are ordered, immutable `golang-migrate` SQL migrations
recorded in the database migration ledger. A separate one-shot migration command
applies them before API replicas start. The long-running API must validate and use
the resulting schema; it must never create, alter, or migrate schema at startup.

Local Compose uses durable named PostgreSQL storage. SpiceDB uses its PostgreSQL
datastore rather than the ephemeral testing server, runs its datastore migration
as an explicit one-shot dependency, and authenticates API gRPC requests even on
the explicitly plaintext loopback development transport. Resource and relationship
state must remain aligned across complete stack restarts.
PostgreSQL health remains false while its image entrypoint runs the temporary
initialization server. Dependent migrations start only after the final PostgreSQL
process is PID 1 and accepts connections, including on a brand-new volume.
Compose runs the same non-root adapter-node web image used for production. Its
build consumes the frozen workspace lockfile and deploys production
dependencies, so local acceptance exercises the deployable artifact.
Local Compose uses ordinary bridge networking and binds every published port to
`127.0.0.1`; host networking is not a prerequisite. OIDC discovery and key
retrieval may use a separately validated internal backchannel base URL while
signature, issuer, and audience validation continue to use the browser-visible
issuer. None of these services accepts LAN connections despite reviewed
development credentials. Compose loads `.env` for policy configuration and
overrides only topology-specific bind addresses and internal service endpoints.
The host-published SpiceDB port belongs to a typed runtime-capability proxy, not
the upstream SpiceDB server. Its bearer credential is unable to invoke schema
mutation RPCs.

`make dev` starts the stateful infrastructure in containers and runs the Go API
and SvelteKit development server on the host with hot reload and clean signal
handling. Production-like Compose continues to run the same non-root API and web
images used by release acceptance.

Authorization schema application is a separate, one-shot bootstrap operation.
The long-running API process never writes authorization policy during startup and
does not require schema-administration behavior. Deployment must run the schema
job before API replicas start and may provide it distinct credentials.

Authorization relationship changes use a transactional outbox. Workers claim
bounded batches with an owner token and expiring lease before contacting SpiceDB.
Only the lease owner may complete or fail a claim. This prevents concurrent API
replicas from processing the same change while allowing abandoned work to recover.
Workers renew the claim only after obtaining the per-resource serializer and
must not contact SpiceDB when renewal proves that their lease is stale.
PostgreSQL is the sole time authority for authorization outbox leases so clock
skew between API replicas cannot steal, extend, or complete another claim.
Create responses are not successful until their owner relationship has been
written; durable pending work remains retryable when SpiceDB is unavailable.

# Platform Architecture Specification

## Structure

The API uses hexagonal architecture. Domain packages contain business concepts;
application packages coordinate use cases; ports describe required capabilities;
adapters implement transport, persistence, identity, authorization, and telemetry;
bootstrap composes adapters and owns lifecycle.

## Baseline stack

- Go API with Huma-generated OpenAPI.
- PostgreSQL with GORM and versioned migrations.
- OIDC authentication with immutable issuer/subject identity mapping.
- SpiceDB authorization behind a project-owned port.
- TypeScript contracts generated with pinned `openapi-typescript` and consumed
  using pinned `openapi-fetch`.
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

All runtime configuration is environment-backed and validated at startup.
Infrastructure dependencies are replaceable adapters.
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

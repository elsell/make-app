# Audit Specification

Audit is a mandatory platform bounded context. It records durable facts about
authenticated application behavior independently of operational logs.

Each event contains an immutable event ID, visibility owner user ID, actor user
ID, typed action, target type and ID, outcome, correlation ID, and UTC occurrence
time. Baseline actions are `user.provisioned`, `resource.listed`,
`resource.viewed`, `resource.created`, `resource.updated`, `resource.deleted`,
and `resource.access_denied`. Authorization reconciliation emits
`authorization.relationship_applied` or
`authorization.relationship_failed`. Outcomes are `succeeded`, `denied`, and
`failed`. A successful resource mutation event means its domain and outbox
transaction committed; the separate authorization event states whether the
corresponding SpiceDB write succeeded before the API response.

User provisioning and successful resource mutations write their audit event in
the same PostgreSQL transaction as the state change. Authenticated reads and
denials append their event before responding. Audit persistence failure fails
the operation closed. Events never contain bearer tokens, credentials, complete
request bodies, arbitrary caller metadata, IP addresses, or user-agent strings.

`GET /v1/audit-events` returns a signed cursor-paginated stream containing only
events performed by the current user or affecting resources owned by the current
user. A caller cannot select another principal. The initial stream is ordered by
immutable `(occurred_at, id)` and excludes events inserted after traversal began.
Listing audit history does not recursively create another audit event; otherwise
observing the stream would mutate it and make complete pagination impossible.

Application ports expose append and list only. PostgreSQL rejects updates and
deletes and truncation with append-only triggers. The runtime database role has
only `SELECT` and `INSERT` on audit history and cannot mutate the migration
ledger; the distinct migration owner applies DDL. Audit-producing operations
are bounded per authenticated principal with an environment-configured,
memory-bounded limiter. Production deployments must additionally alert on audit
write rate and database capacity and scale the limiter across replicas. Future
retention, legal hold, export,
tamper-evident hashing, administrative access, or external audit sinks require
explicit specifications and privileged adapters.

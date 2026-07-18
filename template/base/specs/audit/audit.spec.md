# Audit Specification

Audit is a mandatory platform bounded context. It records durable facts about
authenticated application behavior independently of operational logs.

Each event contains an immutable event ID, visibility owner user ID, actor user
ID, typed action, target type and ID, outcome, correlation ID, and UTC occurrence
time. Baseline actions include `user.provisioned`, `user.viewed`, `resource.listed`,
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
are bounded per authenticated principal with a PostgreSQL-coordinated limiter
shared across replicas. Production deployments must additionally alert on audit
write rate and database capacity.

Retention is disabled unless an operator supplies a separate retention DSN,
period of 30–3650 days, and batch size of 1–10,000. The retention role cannot
read, insert, update, or directly delete detail audit rows. It may only append a
bounded retention request; a reviewed security-definer trigger validates that
the cutoff is at least 30 days old, overwrites the untrusted completion time,
locks a bounded set, performs the delete, and records the actual count in the
same database statement. This database primitive is used through ordinary GORM
model creation, avoiding an application-side raw-SQL exception. Dry runs append
the same immutable record but only count eligible rows. The API role remains
unable to access retention records or delete audit details. Retention records
cannot be updated, deleted, or truncated, and are not themselves subject to the
detail-event retention command. The worker stops cleanly between batches and
emits typed operational probes. Legal hold, export, tamper-evident hashing,
administrative event browsing, or external audit sinks require further specs.
The detail append-only trigger permits deletion only when PostgreSQL reports the
retention credential as `session_user` and the migration-owned security-definer
function as `current_user`; direct use of either role remains denied.

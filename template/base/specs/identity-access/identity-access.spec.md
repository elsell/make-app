# Identity and Access Specification

## Authentication

The API accepts provider-signed OIDC ID tokens only at the session-exchange
boundary. Tokens must have a valid signature, issuer, audience, expiry, and
non-empty subject. A successful exchange returns a cryptographically random,
opaque application session credential. All other API routes reject OIDC tokens
and accept only application sessions. Only a SHA-256 digest of each session
credential is stored. Sessions are scoped to interactive user API access, have
a configurable bounded lifetime, can be revoked, and fail closed for disabled
or deleted accounts. Provider access and refresh tokens are never accepted by,
stored in, or relayed to ordinary application routes.
An identity is uniquely keyed by issuer and subject. First exchange creates a
local user idempotently. Later verified exchanges synchronize provider email and
display-name claims without changing the stable local identity. Provider claims
are mutable profile data and never authorization identifiers. Account status is
an explicit typed lifecycle value. Disabled accounts cannot exchange or use a
session. Account disablement revokes all active sessions transactionally.
When `ACCOUNT_SELF_DEACTIVATION_ENABLED` is true, `DELETE /v1/me` disables the
authenticated account, revokes every session, and appends `user.deactivated` in
the same database transaction. It is disabled by configuration otherwise and
returns 403 without changing state. The baseline retains the disabled identity,
resources, and audit history; permanent erasure is product- and retention-policy
specific and is not implied by this operation.
Account provisioning mode is explicit configuration: `open` allows a verified
OIDC identity to create its local account, while `existing` permits only an
already provisioned or invitation-claimed identity. The baseline's invitation
adapter is a normalized, deduplicated environment-backed email allowlist. A
provider email on that list may provision as an invited user only when the OIDC
provider explicitly marks the email verified;
unlisted identities remain rejected. Normalized non-empty local emails are
database-unique, so one invitation cannot provision multiple OIDC subjects even
under concurrent exchange. Products may replace this adapter with a
durable invitation application port without changing the stable session or
authorization identity. The baseline never silently falls back from `existing`
to unrestricted self-registration.

`GET /v1/me` returns the authenticated local user.
First-time local user provisioning writes `user.provisioned` in the same
transaction as the user and external identity. Routine `/v1/me` calls do not
pretend that the identity provider's own authentication ceremony occurred at
the API boundary.
Changed OIDC email or display-name claims write `user.profile_synchronized` in
the same transaction as the profile update. Application session creation and
revocation write `session.created` and `session.revoked` atomically with their
session row changes; audit targets use the user identity and never store bearer
credentials or credential hashes.

Authentication failures return 401. Authorization denials for resource IDs
return 404 to avoid disclosing existence. Infrastructure and persistence errors
must never be mislabeled as authentication failures or resource absence; they
return a generic 500 response without exposing internal details.

Browser and native clients use authorization code with PKCE. They hold the OIDC
ID token only long enough to exchange it and retain only the application session
credential afterward. Browser sessions use session-scoped storage rather than
persistent local storage; native sessions use platform secure storage. Clients
clear rejected or expired sessions, call `/v1/me` after exchange, and surface
failures without rendering a stale authenticated state.
The bundled local Dex explicitly allows the generated web origin for its public
PKCE client's cross-origin token exchange. Production providers must likewise
allow the deployed web redirect origin; no client secret is embedded in the web
application.
Before expiry, clients call the authenticated session-refresh route. Refresh
atomically revokes the presented application credential, creates a new opaque
credential with a fresh bounded expiry, and writes both session audit events in
one database transaction. The old credential cannot be reused after rotation.
Web and native adapters schedule the next rotation while the application remains
open and also refresh near-expiry credentials during restored-session startup.
Every rotation preserves the session family's original absolute expiry. The
absolute lifetime is environment-configurable, must be at least the rotating
session lifetime, and cannot exceed seven days. A requested rotation is capped
at that deadline; once it is reached, the user must authenticate with OIDC again.
Web and mobile treat a non-advancing replacement expiry as the end of the
renewable family: they retain that replacement only until its stated expiry,
then clear it and require OIDC authentication. Clients must not rapidly rotate
inside the final refresh window; short configured session lifetimes use a
bounded midpoint refresh instead.

The generated interactive API documentation uses the same OIDC discovery
metadata and a dedicated public documentation client. Its authorization flow is
authorization code with PKCE, never an embedded client secret. Protected
operations expose an authorization control, and an authenticated documentation
session can invoke `/v1/me` and protected resource routes.
The API obtains the authorization and token endpoints from provider discovery;
it never derives them by appending assumed paths to the issuer URL.
The documentation page uses a reviewed, versioned Scalar asset protected by
subresource integrity. Its restrictive CSP permits OIDC discovery and token
exchange only through fixed same-origin relays; the required Scalar
inline-style and evaluation exceptions are isolated to this documentation page.
Because providers do not consistently allow browser CORS on discovery and token
endpoints, docs expose fixed same-origin discovery and token relay routes. They
proxy only the configured issuer, use bounded requests/responses and timeouts,
reject all upstream redirects so credentials and PKCE material cannot cross an
origin boundary,
rewrite no authorization endpoint, and never hold or add a client secret.
The token relay accepts Scalar's public-client Basic-auth request shape but
forwards no authorization header. It validates an authorization-code PKCE
exchange and supplies only the configured documentation client ID and redirect
URI, ignoring caller attempts to override either identity.
The documentation security scheme is rendered as OAuth 2 authorization code in
OpenAPI because the pinned Scalar renderer does not propagate client and PKCE
settings from an `openIdConnect` discovery scheme. It requests OIDC scopes and
the fixed token relay exchanges the verified OIDC ID token for a short-lived
application session before returning it to Scalar. It never returns provider
access, refresh, or ID tokens to the documentation application.

Production OIDC discovery and SpiceDB transport require TLS. PostgreSQL requires
certificate and hostname verification. Plaintext local development is available
only through separate, explicit insecure flags; secure mode is the default and
startup rejects contradictory or malformed configuration.

## Authorization

SpiceDB decides permissions for protected resources. The initial model gives a
user `owner` relation to their own resources. Create operations establish that
relationship; reads and changes check permission and fail closed. Lists are
scoped by authenticated user in persistence and must not use SpiceDB as a broad
resource-discovery mechanism.
Runtime relationship/readiness access and schema administration use distinct
configured credentials. The schema command accepts only the schema credential;
the API never receives it. Local Compose exposes SpiceDB only to its private
network and places a typed authenticated gRPC capability proxy in front of it.
The runtime credential can call only relationship writes, permission checks, and
schema reads; schema writes and all other RPCs fail with permission denied. Only
the proxy and one-shot schema job receive the upstream administration credential.

Authorization outbox leases control claim ownership, while a PostgreSQL-backed
per-resource serializer is held across each SpiceDB relationship write. Lease
expiry may cause an idempotent retry, but it must never allow TOUCH and DELETE
to execute out of order or let a delayed worker resurrect access. Completion
and failure updates require both the current lease owner and an unexpired lease.
After acquiring the serializer and before contacting SpiceDB, a worker must
atomically renew its still-owned, still-unexpired claim for longer than the
bounded SpiceDB call. A stale or reclaimed worker must stop without making an
external authorization write.
Claim, renewal, expiry, completion, and failure predicates use PostgreSQL's
clock, never a replica's process clock. The narrowly scoped GORM timestamp and
interval expressions are an explicit exception to the general raw-SQL ban
because database time is the lease-fencing authority across replicas.
Failed authorization changes retry only up to the configured attempt limit.
The final failure atomically marks the item dead-lettered with a bounded typed
failure code and no provider error text. Dead letters are never claimed
automatically and continue to fence later changes for the same resource. An
authenticated affected resource owner can list only dead letters whose subject
is that user through the same signed, stable cursor pagination used by other
collections and explicitly requeue one through the API. Requeue atomically clears
the dead-letter state, preclaims the exact change for the requesting worker, and
appends `authorization.dead_letter_requeued`; a background worker cannot steal
the recovery between those operations. Requeue never changes the original
relationship payload. Cross-user identifiers are concealed as not found.

Tenant, team, organization, and sharing concepts are not part of the baseline.
They require product specs before introduction.

Denied authenticated resource decisions are audited against the requesting
actor without revealing the target owner or existence. A failure to persist the
denial event fails closed as an internal error. Unauthenticated failures are not
stored in the domain audit stream because no trustworthy actor exists; operators
observe them through security telemetry.

## Idempotent Commands

Every generated create route requires an `Idempotency-Key` with 16 to 128
printable ASCII characters. The persistence transaction reserves the tuple of
authenticated principal, operation, and key together with a canonical request
digest. A retry with the same digest returns the original resource and does not
repeat the mutation, authorization outbox write, or creation audit. Reusing the
key with a different request fails with 409. Idempotency is enforced by a
database uniqueness constraint and is safe across processes and replicas.

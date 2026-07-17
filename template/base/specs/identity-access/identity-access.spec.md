# Identity and Access Specification

## Authentication

The API accepts OIDC bearer tokens validated using provider discovery. Tokens
must have a valid signature, issuer, audience, expiry, and non-empty subject.
The bearer credential is the provider-signed OIDC ID token; browser and native
clients must not send an opaque OAuth access token to the ID-token verifier.
An identity is uniquely keyed by issuer and subject. First authentication creates
a local user idempotently. Provider email and display-name claims are mutable
profile data and never authorization identifiers.

`GET /v1/me` returns the authenticated local user.

Authentication failures return 401. Authorization denials for resource IDs
return 404 to avoid disclosing existence. Infrastructure and persistence errors
must never be mislabeled as authentication failures or resource absence; they
return a generic 500 response without exposing internal details.

Browser and native clients use authorization code with PKCE. Browser tokens use
session-scoped storage rather than persistent local storage. Both the initiating
page and callback share one exact OIDC configuration. Clients request refresh
capability, renew before expiry where the platform permits it, clear invalid or
expired sessions, call `/v1/me` after sign-in, and surface failures without
rendering a stale authenticated state.

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
The documentation security scheme is rendered as OAuth 2 authorization code in
OpenAPI because the pinned Scalar renderer does not propagate client and PKCE
settings from an `openIdConnect` discovery scheme. It requests OIDC scopes and
the API still accepts only verified OIDC ID tokens.

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

Tenant, team, organization, and sharing concepts are not part of the baseline.
They require product specs before introduction.

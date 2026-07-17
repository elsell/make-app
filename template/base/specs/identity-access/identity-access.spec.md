# Identity and Access Specification

## Authentication

The API accepts OIDC bearer tokens validated using provider discovery. Tokens
must have a valid signature, issuer, audience, expiry, and non-empty subject.
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

Tenant, team, organization, and sharing concepts are not part of the baseline.
They require product specs before introduction.

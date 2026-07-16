# Identity and Access Specification

## Authentication

The API accepts OIDC bearer tokens validated using provider discovery. Tokens
must have a valid signature, issuer, audience, expiry, and non-empty subject.
An identity is uniquely keyed by issuer and subject. First authentication creates
a local user idempotently. Provider email and display-name claims are mutable
profile data and never authorization identifiers.

`GET /v1/me` returns the authenticated local user.

## Authorization

SpiceDB decides permissions for protected resources. The initial model gives a
user `owner` relation to their own resources. Create operations establish that
relationship; reads and changes check permission and fail closed. Lists are
scoped by authenticated user in persistence and must not use SpiceDB as a broad
resource-discovery mechanism.

Tenant, team, organization, and sharing concepts are not part of the baseline.
They require product specs before introduction.


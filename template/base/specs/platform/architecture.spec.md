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

All runtime configuration is environment-backed and validated at startup.
Infrastructure dependencies are replaceable adapters.

Database schema changes are ordered, immutable migrations recorded in the
database migration ledger. Startup may apply unapplied migrations transactionally;
application bootstrap must never perform unversioned schema mutation.

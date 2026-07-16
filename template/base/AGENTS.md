# __APP_NAME__ Engineering Guidance

This is a spec-driven, security-conscious application. Specifications under
`specs/` are the source of truth. Update the relevant `*.spec.md` before coding.

## Architecture

- Always use hexagonal architecture with explicit domain, application, port,
  adapter, and bootstrap boundaries.
- Domain code must not import HTTP, persistence, OIDC, SpiceDB, framework, or
  observability implementations.
- Use dependency injection. Keep command entrypoints thin.
- Organize HTTP adapters domain-first; separate routes, DTOs, and mapping when a
  surface grows beyond trivial behavior.
- GORM models are infrastructure mappers. Domain entities never persist themselves.
- User-owned repository reads require a user ID. Unscoped reads are a security smell.

## Specifications and documentation

- Product specs live in domain directories as `specs/<domain>/*.spec.md`.
- Cross-cutting engineering specs live under `specs/platform/`.
- Code and docs follow specs. Resolve disagreements by updating the spec first.
- Keep human documentation concise, verified, and focused on useful workflows.

## Identity, authorization, and tenancy

- OIDC identities use immutable `(issuer, subject)` keys. Email is profile data.
- Authentication and authorization use separate ports.
- SpiceDB is the permission decision point for protected resources.
- Do not assume organizations or tenants. Add them only through a product spec.
- Model ownership and sharing explicitly in domain language and authorization schema.

## Security

- Pin dependencies, actions, tools, and container images. Runtime images require
  immutable digests; floating tags such as `latest` are forbidden.
- Every authentication or authorization boundary needs adversarial end-to-end
  coverage: unauthenticated, malformed, expired, wrong issuer/audience,
  cross-user, insufficient permission, dependency failure, and legitimate access.
- Fail closed and avoid leaking whether inaccessible resource identifiers exist.

## Testing

- Use TDD: failing behavioral test, smallest implementation, then refactor.
- Never use mocks. Use realistic in-memory or controlled fakes behind ports.
- Test functionality through appropriate ports and real interaction boundaries.
- Inject clocks for behavior involving time, leases, audit, expiration, or tests.

## Observability and configuration

- Use typed, domain-oriented events through injected, fan-out-capable ports.
- Do not add `print`, `println`, or ad hoc logging in application code.
- Read environment variables only in configuration/bootstrap packages.
- Validate configuration before starting network listeners or workers.

## API contracts and clients

- Use Huma-generated OpenAPI as the REST contract.
- Keep envelopes and pagination consistent.
- Generate TypeScript contracts with pinned `openapi-typescript`; access them
  through pinned `openapi-fetch` behind frontend adapters.
- CI must reject stale generated output.

## Workflow

1. Update the relevant spec.
2. Write the failing test with fakes.
3. Implement through ports and adapters.
4. Add adversarial security tests for boundaries touched.
5. Run formatting, tests, contract drift checks, and structural checks.
6. Update the roadmap only when focus, sequencing, evidence, or blockers change.
7. Commit atomically using a Conventional Commit message when asked.


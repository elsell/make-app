# Make App Engineering Guidance

This repository is spec-driven. Update `specs/*.spec.md` before behavior or
architecture changes, write a failing test, implement the smallest correct
behavior, and keep documentation synchronized.

## Architecture

- Use hexagonal architecture. Domain logic must not depend on HTTP, databases,
  identity providers, SpiceDB, frameworks, or observability implementations.
- Express infrastructure through narrow ports and inject adapters at bootstrap.
- Keep commands thin and split bootstrap code by startup responsibility.
- Organize HTTP adapters domain-first into routes, DTOs, and mappers.
- Treat persistence models as infrastructure mappers, never domain entities.
- Repository reads for user-owned data must require the owner ID. Never add an
  unscoped read for convenience.

## Identity and security

- Authenticate OIDC identities by immutable issuer and subject, not email.
- Keep authentication and authorization distinct and behind separate ports.
- Make SpiceDB the permission decision point for protected resources.
- Test boundaries adversarially: missing, malformed, expired, wrong-issuer and
  wrong-audience tokens; unauthorized IDs; cross-user access; and valid access.
- Fail closed when an authorization dependency is unavailable.
- Pin dependencies, actions, tools, and container images to reviewed versions.
  Runtime container images must use immutable digests.
- Run the fourteen-day dependency-age gate for every dependency change. Exact,
  reviewed exceptions require a spec update and `dependency-age-allowlist.json`
  entry with a reason and compensating verification.

## Testing and operations

- Use test-driven development. Use behavioral fakes, never mocks.
- Use an injected clock whenever time affects behavior or tests.
- Emit typed, domain-oriented events through an injected observability port.
  Do not use ad hoc print statements in application code.
- Parse and validate environment configuration only at application boundaries.
- Keep OpenAPI generated from the API implementation and generated clients in
  sync. CI must reject drift.
- Turn recurring structural mistakes into automated checks.

## Workflow

Run formatting, tests, generated-contract checks, and relevant integration tests
before finalizing. Use atomic Conventional Commits when committing.

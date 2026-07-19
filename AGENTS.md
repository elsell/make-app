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
- Use Red Hat Hardened Images wherever the catalog supplies a compatible build
  or runtime component. Pin an immutable release tag and multi-platform digest,
  review its SBOM and CVE report, and verify Red Hat's signature when updating
  it. Use a reviewed upstream image only when no compatible HI component exists.
  Selecting a FIPS-tagged image alone does not establish FIPS compliance.
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
- Treat generated-repository adoption as a transactional migration. Preserve Git
  history and user specs, merge only declared guidance surfaces, and refuse all
  ambiguous conflicts before writing.
- Describe mobile evidence precisely: export, prebuild, Gradle/Xcode compilation,
  development client, signing, and publication are different guarantees.
- Native session clients must retain valid credentials across transient network,
  rate-limit, and service failures. Clear only expired, explicitly rejected,
  revoked, or unreadable credentials.
- Native cold-launch restoration must not wait for OIDC discovery. Production
  web images must fail closed on missing or unsafe public API and OIDC settings;
  loopback defaults require an explicitly selected development environment.
- Keep Apache-2.0 licensing and public security/compatibility metadata in both the
  generator and every generated repository.

## Workflow

Run formatting, tests, generated-contract checks, and relevant integration tests
before finalizing. Use atomic Conventional Commits when committing.

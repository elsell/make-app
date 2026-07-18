# Platform Roadmap Specification

## Current focus

Bootstrap the first product bounded context on the generated identity, session,
authorization, audit, observability, persistence, web, and mobile platform.

## Required evidence

- Keep this file current only when focus, sequencing, completion evidence, or a
  known blocker changes.
- Record the most recent full verification and adversarial boundary acceptance
  when a roadmap item is completed.

## Completion evidence

- 2026-07-18: the generated platform baseline passed `make verify` and full
  `make acceptance` for example and blank projects. Evidence included a
  generated domain's fail-closed authentication/policy HTTP boundary, OpenAPI
  registration, PostgreSQL and SpiceDB integration, browser OIDC and application
  sessions, Scalar OIDC Try It, audit immutability, dependency failure probes,
  production web/API images, and iOS/Android Expo exports.

## Next decisions

- Specify the first product application service and its SpiceDB relationships.
- Specify sharing roles and authorization-scoped shared-resource listing before
  adding viewer/editor behavior.

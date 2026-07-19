# Production checklist

Deploy the API and web images by immutable digest. Run PostgreSQL migrations and
the SpiceDB datastore migration/schema jobs with separate credentials before
starting API replicas. API database credentials must not own schema or migration
state; API SpiceDB credentials go through the capability proxy and cannot change
schema.

Required preparation:

- Set `PUBLIC_APP_ENV=production`, `PUBLIC_API_URL`, `PUBLIC_OIDC_ISSUER`, and
  `PUBLIC_OIDC_CLIENT_ID` for the web container. The image defaults to production
  mode and refuses to serve with absent, local, credentialed, or non-HTTPS
  production endpoints. Local Compose explicitly overrides this to development.
- Keep the generated Red Hat Hardened Images for Go, the static API runtime,
  Node.js, and PostgreSQL unless a documented compatibility constraint requires
  another source. When updating a base, select an immutable catalog release,
  pin its manifest-list digest, inspect its SBOM and CVE report, and verify its
  Red Hat signature using the catalog's current instructions. Hardened Images
  are usable without a Red Hat subscription. Do not claim FIPS compliance from
  a FIPS image tag alone; validate the complete application, cryptographic
  configuration, host, and deployment boundary.
- Terminate TLS at a trusted proxy and forward only validated scheme/host data.
  Keep API, PostgreSQL, SpiceDB, OTLP, and OIDC transport encryption enabled.
  Set `__ENV_PREFIX___TRUSTED_PROXY_CIDRS` to only that proxy network when
  client-source forwarding is required; leave it empty for direct deployments.
- Generate independent high-entropy cursor, metrics, database, SpiceDB, and
  other credentials. Never deploy values from `.env.example` or Compose.
- Configure exact public API/web origins, CORS origins, OIDC clients and redirect
  URIs using `docs/oidc.md`.
- Size the PostgreSQL connection pool across all replicas, not per process in
  isolation. Configure request/audit limits, session lifetimes, and authorization
  retry/dead-letter policy for expected traffic.
- Configure OTLP export, scrape the authenticated metrics endpoint, and alert on
  readiness, authorization dead letters, audit-write rate, database capacity,
  latency, and error probes.
- Choose and test an audit retention period. Retention is disabled by default
  and must run with its separate database role; the API runtime cannot erase
  audit events.
- Bootstrap invitation administrators before selecting invitation-only account
  provisioning, then test invitation expiry, consumption, revocation, and
  concurrent first login.
- Route liveness and readiness separately. During rolling deployment, complete
  migrations and authorization schema first, wait for readiness, and drain API
  replicas using their bounded shutdown window.
- Exercise restore procedures and a complete staging rollout even when backup
  infrastructure is managed outside this repository.

Scalar documentation can be disabled. Do so unless interactive production API
documentation is intentional and its public OIDC client is registered.

# Make App Generator Specification

## Purpose

`make-app` bootstraps production-shaped application monorepos with a Go API,
PostgreSQL, SpiceDB, OIDC, generated TypeScript API contracts, a web client, and
an Expo native client. Generated applications own their source code and do not
depend on a Make App runtime framework.

## Commands

- `make-app version` prints the installed module tag, or `dev` for an untagged
  development build, so issue reports and adoption records identify the generator.

- `make-app doctor` checks every required local tool, its supported version,
  Docker Compose availability, and the local ports used by the generated
  development topology. It reports all failures in one run and never mutates the
  workstation.
- `make-app new NAME --module MODULE [--bundle-prefix PREFIX] [--output DIR] [--without-example]` creates
  a new repository through a sibling staging directory and atomically renames it
  into place only after rendering, formatting, Git initialization, hook
  installation, and manifest creation succeed. Failures leave no destination
  that prevents an identical retry. Generated repositories always use `main` as
  their initial branch. NAME is constrained to the generator's safe display-name
  grammar, and MODULE must pass Go's canonical module-path validation before any
  staging directory or destination is written.
- `make-app init NAME --module MODULE [--bundle-prefix PREFIX] [--dir DIR]
  [--without-example]` adopts an existing spec-first Git repository without
  replacing its history. Git itself must validate the destination as the exact
  worktree root, including linked worktrees; the destination must not already
  contain `.make-app.json`, and may initially contain only `.git`, `AGENTS.md`,
  `README.md`, `.gitignore`, `LICENSE`, `specs/`, `docs/`, and `.codex/`.
  Generation happens in a sibling staging directory. Existing guidance and
  documentation are preserved; generated guidance is appended in an explicitly
  delimited section, ignore rules are unioned, and directory trees are merged
  only when every overlapping file is byte-identical. A conflicting license,
  spec, document, or agent definition is refused before mutation with the exact
  path. Symlinks in adoptable content are refused before reads or writes, so
  generated files cannot escape the repository. Git is never reinitialized.
  Hook paths are resolved through Git so configured `core.hooksPath` values are
  honored only inside the worktree or that repository's Git common directory;
  external and symlink-escaping paths are refused. Installation is rollback-safe and leaves the existing repository
  byte-for-byte unchanged on failure.
- Mobile bundle, Android package, and URI-scheme identifiers use a separate
  deterministic, ASCII alphanumeric, letter-leading native identifier derived
  from the application slug; display names and repository slugs are not reused
  where native platform grammars are stricter.
- `--bundle-prefix` accepts a validated lowercase reverse-DNS prefix and renders
  both iOS and Android application identifiers. Its local default is
  `com.example`; production apps should set their owned organization prefix at
  generation time, and the value is persisted in `.make-app.json`.
- Environment prefixes are independently normalized to a shell-safe,
  letter-leading identifier. Domain and REST identifiers are capped at 40 ASCII
  characters so generated PostgreSQL identifiers remain below its 63-byte limit.
- `make-app domain add NAME [--dir DIR] [--plural PLURAL] [--fields SPEC]` adds a user-owned vertical-slice
  scaffold. It generates a typed entity, application repository and HTTP service
  ports, domain-owned GORM repository/model, dedicated PostgreSQL table migration,
  DTOs, mappers, Huma route registrar, and focused tests. It never registers the
  new domain against the example domain's shared storage or generic routes. The
  developer writes the application service and explicitly wires the registrar
  after specifying behavior and sharing.
- Generated create and delete repositories populate authorization outbox owner
  and actor fields independently from the relationship subject, and their real
  PostgreSQL test rejects missing or misattributed authorization context.
- `--plural` overrides the REST collection identifier and route plural; the
  dedicated PostgreSQL table remains named from the singular domain concept.
  Without it, the generator handles
  common English `-y`, `-s`, `-x`, `-z`, `-ch`, and `-sh` forms rather than
  blindly appending `s`.
- Domain REST collections must be unique in the project manifest and cannot use
  platform-owned `/v1` collections such as `me`, `sessions`, `invitations`, or
  `audit_events`.
- `--fields` is a comma-separated list of `name:type` declarations. Supported
  initial types are `string`, `bool`, `int`, `float`, and `time`; identifiers,
  ownership, and timestamps remain platform fields. The same typed field set is
  rendered consistently into the entity, migration, persistence mapper, DTOs,
  mapper, and route service contract. Omitting it preserves the useful
  `name:string` starter field.
- A generated slice accepts at most 25 fields, and field identifiers are capped
  at 40 ASCII characters. This keeps PostgreSQL identifiers unambiguous and all
  generated Go files within the enforced structural size limit.
- Generated update routes use `PUT` with every declared domain field required,
  making full-replacement semantics explicit. The generator does not emit a
  `PATCH` contract that can silently zero omitted fields.
- A successful domain addition prints exact remaining application-service,
  registration, OpenAPI, client, authorization, audit, and test steps. It runs
  formatting and contract generation when dependencies are already installed,
  while clearly reporting when regeneration is deferred until bootstrap.
- `make-app example remove [--dir DIR]` removes the demonstration domain only
  when no user code outside the generator-owned removable surfaces imports the
  example domain or calls its REST collection. It refuses before mutation when a
  dependency is found, removes its route registration and source surface, and
  adds a forward migration that removes its table. It is failure-atomic and
  restores original content and permissions if a later step fails. New projects
  may omit the example from the start.
- Added-domain migrations use the next unused monotonic migration version and
  never overwrite or reuse an existing version.
- Domain addition is failure-atomic: generated files are formatted in staging,
  and any failed installation, metadata update, or eager contract generation
  removes staged additions and restores modified generator metadata and generated
  client contracts byte-for-byte so the same command can be retried.
- Generation refuses to overwrite non-empty destinations or existing domains.
- `.make-app.json` records the generator/template schema version, application
  identity, module, domains, and their explicit plurals and fields. Mutating
  commands reject incompatible projects with an actionable upgrade message;
  they never apply a new scaffold to an unknown older layout.
- Automatic domain DI and route composition define template schema version 4;
  mutating a version-3 project with the new layout is rejected rather than
  silently producing an unregistered domain. The rejection links to the exact,
  repository-owned v3-to-v4 upgrade procedure; changing only the manifest
  version is explicitly unsupported. The procedure preserves the complete
  numeric migration sequence, including product migrations interleaved between
  generated domain migrations, and verifies the final migration version.

## Bootstrap and developer experience

- `make bootstrap` is the canonical fresh-start command. It creates `.env` from
  the checked-in `.env.example` only when `.env` is absent, never overwrites local
  configuration, and leaves `make dev` immediately usable.

- The documented pinned installation version must exist before it appears in
  the README. The generator release workflow must verify its own Go-only root
  layout and publish the first usable tag; it must not assume the generated
  JavaScript workspace exists in the generator repository.
- Generated `make bootstrap` verifies prerequisites, installs dependencies,
  generates contracts, and prints the local URLs and next commands.
- `.env` is an actual runtime input. Host development loads it directly;
  Compose loads configurable policy values from it while overriding only the
  topology-specific bind addresses and internal service endpoints.
- Default Compose uses portable bridge networking. Host networking is not a
  prerequisite. OIDC supports a separately validated internal backchannel URL
  while preserving the browser-visible issuer for issuer validation.
- `make dev` starts infrastructure in containers and runs the Go API and
  SvelteKit web application on the host with hot reload and clean signal
  handling. Focused targets provide logs, database shell, migration, reset,
  seed, web, API, and Expo mobile workflows.
- Mobile documentation and checks cover iOS simulator, Android emulator, and
  physical-device API/OIDC URLs, public-client redirect registration, Expo
  development builds, and the limits of Expo Go custom-scheme authentication.
- Human documentation includes verified development, domain-completion, OIDC,
  mobile, and production-deployment guides.
- `make-app doctor` reports that native macOS and Linux are the supported hosts
  and that Windows requires WSL2. It distinguishes universal prerequisites from
  host-specific Android and iOS requirements and reports actionable native-tool
  limitations without claiming iOS compilation is available on non-macOS hosts.
  On macOS it checks CocoaPods for iOS builds; on every host it checks a usable
  Android SDK executable as well as Java and SDK environment configuration.

## Native mobile delivery

- Expo bundle export, clean prebuild validation, unsigned native compilation,
  development-client creation, and signed store release are distinct commands
  and CI evidence. `mobile:export` remains the fast JavaScript bundle check; it
  is never described as a native build.
- Generated mobile dependencies include the Expo development client. Checked
  `eas.json` profiles provide development, preview, and production builds while
  local `expo run:ios` and `expo run:android` commands remain available without
  requiring EAS. Development and production commands set an explicit environment
  profile and cannot silently consume the other profile's public endpoints.
- Native identifiers, the custom OIDC callback scheme, Expo slug, iOS bundle ID,
  and Android package are rendered from the same validated project identity.
  A checked validation script rejects drift before prebuild or release.
- The generated Expo configuration includes application version, iOS build
  number, Android version code, runtime-version policy, update policy, orientation,
  icons, adaptive icons, splash assets, and user-replaceable placeholders.
  Release documentation covers permissions, credentials, signing, store metadata,
  and provider callback registration without committing secrets.
- CI runs Expo Doctor, dependency compatibility checks, deterministic clean
  prebuilds, and an unsigned Android Gradle compilation on Linux. A separate
  macOS job performs clean iOS prebuild and unsigned simulator compilation on
  pushes to `main`, releases, and manual dispatch. Pull requests retain the fast
  platform-neutral validation while avoiding untrusted signing or credentials.
  Android native CI installs a pinned JDK distribution before Gradle compilation.
- If Expo's install-check metadata disagrees with the exact React Native version
  in the installed Expo package's `bundledNativeModules.json`, the template may
  exclude only `react-native` from that stale metadata check. A repository-owned
  validator must require an exact match to the installed Expo manifest; no other
  native dependency may use this exception.
- Mobile session handling is a tested state machine shared through
  `packages/client-core`. Transient network errors, rate limits, and server
  unavailability retain a locally valid application credential and produce an
  authenticated-offline state. Only expiry, explicit 401 credential rejection,
  revocation, or unreadable local credential storage removes the credential.
  Refresh and `/v1/me` failures preserve this classification; an unavailable
  network must never be presented as session expiry.
- `packages/client-core` contains framework-independent session state, API error
  classification, clocks, identifiers, retry decisions, and future client use
  cases. It imports neither Svelte nor React Native. Web and mobile consume it
  through presentation-specific adapters and retain separate UI models.

## Distribution and compatibility

- The generator and every generated repository are licensed under Apache-2.0.
  Generated repositories include the license file from their first commit.
- Public-project hygiene includes `SECURITY.md`, a supported-version policy,
  template-schema/generator compatibility table, contribution issue forms that
  request generator version, schema, host OS, and reproduction commands, and an
  explicit `v0.x` stability designation.
- Generated repositories include grouped Dependabot updates for Go modules,
  pnpm, GitHub Actions, Docker, and the Expo/React Native dependency family.
  Automated dependency changes remain subject to the age, vulnerability, lock,
  generated-contract, native-validation, and acceptance gates.
- EAS CLI executes from a dedicated private workspace in the reviewed frozen
  lockfile while retaining `apps/mobile` as its working directory; release
  commands never resolve an executable graph dynamically.
- Native build inputs are reviewed and explicit: local CocoaPods is Bundler-
  locked, CI selects an exact JDK patch, and signed EAS profiles select named
  SDK-compatible Android and iOS images.
- Hosted iOS compilation selects GitHub's explicit `macos-26` runner so Expo
  SDK 55 is validated with Xcode 26 or newer; platform-neutral generation may
  continue on the older supported macOS matrix image.
- Locked Ruby gems participate in the package-age gate and OSV vulnerability
  scan. Generated checks start the real installed EAS binary, not only a fake.
- Production API and OIDC base URLs reject credentials, local hosts, query
  strings, and fragments. Exchanged credentials are validated before storage.
- Frontend session adapters retry only retryable failures and preserve an
  authenticated-offline presentation without relabeling ordinary 4xx failures.
- The generator's hosted acceptance matrix publishes auditable named cases for
  default, without-example, every field type, explicit irregular plural,
  multiple collision-prone domains, example removal, every supported schema
  upgrade, identifier limits, and existing-repository adoption. Platform-neutral
  generation runs on Linux and macOS; native compilation follows the platform
  policy above. Every successful matrix case compiles and tests the generated API
  and regenerates its OpenAPI document; default generated-project acceptance
  separately verifies all frontend packages and live infrastructure.

## Product-slice ergonomics

- The example remains a genuinely copyable end-to-end slice: API behavior plus
  authenticated, paginated list and idempotent create interactions in web and
  mobile clients, with localized loading, empty, validation, success, and error
  states. It is deliberately simple and removable.
- Domain scaffolding does not pretend to invent product behavior. Each added
  domain receives a typed application-service dependency bundle containing the
  authentication, SpiceDB authorization, repository, audit, clock, probe, ID,
  authorization-worker, and cursor-signing capabilities it can use.
- The generated composition registry constructs each domain repository, injects
  those dependencies, and registers its Huma routes in both the runtime API and
  OpenAPI generation. It must never require hand-edited transport composition
  merely to expose the new contract.
- Generated DTO and operation-wrapper schema names are domain-qualified through
  an injective Go identifier transformation that preserves normalized domain
  separators, so names such as `foo_1` and `foo1` cannot collide in Huma.
  Generator regression coverage must register that exact pair with Huma and
  produce OpenAPI, rather than checking emitted identifier text alone.
- Because the generator cannot infer product authorization policy, the initial
  service implementation authenticates every operation and then fails closed
  with a distinct unavailable result until its policy is implemented. Missing or
  malformed application sessions remain 401 responses; a legitimate session
  cannot gain access from the placeholder. Generated adversarial HTTP tests
  enforce both outcomes.
- Authentication adapters retain their error classification through the
  placeholder: only an invalid application credential becomes 401; database,
  cancellation, and other dependency failures remain server failures.
- The injected authorization capability set includes the outbox and per-resource
  serializer used by the baseline reconciliation lifecycle, not merely the raw
  SpiceDB adapter.

## Operational scalability and lifecycle

- Request and audit limiting use a shared PostgreSQL-backed fixed-window adapter
  by default so adding API replicas cannot multiply configured limits. The port
  remains injectable and an in-memory fake remains available for tests.
- Audit retention is explicit and disabled by default. A separately credentialed
  one-shot retention command can delete events older than a validated retention
  period in bounded batches; the API runtime remains unable to update, delete,
  or truncate audit history. Retention actions emit operational metrics and an
  append-only retention summary before eligible detail rows are removed.
- Invitation-only provisioning uses persistent, expiring, single-use invitation
  records. Configured bootstrap administrators can create, list, and revoke
  invitations through audited endpoints; accepting a verified-email OIDC login
  consumes the matching invitation atomically. Environment email allowlists are
  retained only as an explicit bootstrap mechanism.
- Production documentation covers secrets, TLS and trusted proxies, OIDC client
  registration, migrations, rolling start order, probes, OTLP, rate limiting,
  audit retention, invitation bootstrap, and image-by-digest deployment.
- Pre-commit remains fail-closed for dependency age when dependency manifests or
  lockfiles change, but routine source-only commits run the fast structural,
  formatting, generation, and focused test gate. Pre-push and CI retain the full
  race, vulnerability, age, build, and live acceptance gates.
- Generated CI avoids repeating the same full acceptance suite in both CI and
  release planning for one commit. Release publication consumes a successful CI
  result for the exact SHA and still fails closed when that evidence is absent.
  A privileged `workflow_run` publication accepts only a successful `push` run
  from this repository's `main` branch, never a fork run. Checkout, tags,
  releases, images, and attestations all use the one SHA derived from the
  checked-out commit rather than the release event's default-branch SHA. Manual
  publication likewise refuses any commit other than the current remote `main`.

## Generated baseline

- OIDC identities are keyed by issuer and subject and provision a local user.
- Interactive clients use authorization code with PKCE and an application session
  boundary; OIDC identity tokens are not long-lived API bearer credentials.
  Machine credentials are separate, audience-bound, and least privilege.
- Application-session rotation preserves a configurable, non-extendable family
  deadline of no more than seven days, after which OIDC authentication is required.
- Generated web and mobile clients detect a family-deadline-capped refresh, stop
  rotating that family, and expire it locally at the returned deadline. Short
  session lifetimes use bounded midpoint scheduling instead of a one-second loop.
- Create commands require a principal-and-operation-scoped idempotency key.
  Exact replays return the original result and conflicting reuse is rejected.
  REST resource and invitation creation returns `201 Created`; session/token
  exchanges retain their protocol-appropriate success status.
- Generated domain deletes atomically remove their own row, enqueue the matching
  SpiceDB relationship deletion, and append their audit event.
- Identity profiles have specified claim synchronization, invitation linkage,
  configurable disable/delete lifecycle, and audit behavior.
- Existing-only provisioning is usable on a fresh deployment through a validated,
  normalized environment-backed invited-email allowlist; it never becomes open
  registration when the allowlist is empty.
- `GET /v1/me` returns the authenticated local user.
- PostgreSQL stores users and application resources.
- An append-only audit stream is a mandatory platform primitive. User
  provisioning and every authenticated resource read, list, create, update,
  delete, and denied authorization decision produce a structured audit event.
  Business mutations and their successful audit events commit atomically.
- SpiceDB is the authorization decision point for resource access.
- Local and self-hosted topology places a typed credential-checking gRPC proxy
  before SpiceDB so the API runtime credential cannot write authorization schema;
  the upstream administration credential is absent from the API process.
- PostgreSQL schema changes are reviewed `golang-migrate` files applied by a
  one-shot migration service; API startup has no schema mutation privileges.
- A generated `example` resource demonstrates owner-only create, read, and list.
- Resource lists use stable opaque cursor pagination with a bounded page size,
  deterministic ordering, continuation metadata, and invalid-cursor rejection.
- Huma produces OpenAPI; pinned `openapi-typescript` produces client types.
- Huma's interactive documentation is configured with a dedicated public OIDC
  client and authorization-code PKCE so protected routes are usable from docs;
  authorization and token endpoints come from OIDC discovery rather than
  issuer-path assumptions.
- Interactive documentation can be disabled through validated configuration.
- Web and Expo clients use the generated API package.
  Its authentication adapter must merge bearer authorization into the complete
  request produced by `openapi-fetch`; it must preserve contract headers such
  as idempotency keys rather than replacing them.
- Internationalization is a non-optional presentation-layer invariant. A shared,
  typed locale package supplies web and Expo copy, locale negotiation, fallback,
  interpolation, pluralization, and locale-aware number/date formatting. The
  generated baseline includes complete English and Spanish catalogs.
- User-facing client copy must come from locale catalogs. Generated structural
  checks reject literal JSX/Svelte copy and inconsistent or incomplete catalogs;
  non-visible syntax such as HTTP header object keys is not user-facing copy, and
  both example and blank generated clients must pass the same check without
  weakening detection of literal expression copy;
  every API, health, routing, CORS, and OIDC relay error exposes a stable
  machine-readable code rather than relying on localized strings.
- Docker Compose starts PostgreSQL, SpiceDB, Dex, API, and web services.
  Generated first-party build and runtime stages use Red Hat Hardened Images
  wherever the catalog supplies the required component: the Go builder, static
  API runtime, Node.js builder and runtime, and PostgreSQL. Every reference uses
  an immutable release tag and multi-architecture manifest-list digest. Images
  without a compatible Hardened Images catalog entry, including SpiceDB and
  Dex, remain on reviewed upstream images pinned by version and digest. The
  default images are not described as FIPS-compliant; FIPS is an end-to-end
  application and deployment property, not a base-image label.
  PostgreSQL health contract must remain false during the image entrypoint's
  temporary initialization server and become true only after the final
  PostgreSQL process is PID 1 and accepts connections, preventing first-run
  migration races.
  service runs package installation in non-interactive CI mode so a host-mounted
  workspace with incompatible modules is repaired without a terminal prompt. It
  runs as a configurable unprivileged host UID/GID so generated dependency and
  build artifacts remain removable by the developer. Bootstrap and Compose use
  the same repository-local pnpm store so downloaded content persists with the
  generated workspace. Live acceptance permits a bounded ten-minute cold-start
  window for a slow package registry while still failing closed on readiness.
  Generated Make and acceptance entrypoints derive the web container UID/GID
  from the invoking host user rather than relying on the Compose fallback, so
  non-1000 CI runners and developer accounts retain write access. The container
  supplies writable temporary HOME, XDG cache, and npm prefix/cache directories
  because
  arbitrary numeric UIDs need not have an image passwd entry. Acceptance executes
  the exact pinned image as UID/GID 1001 to prove the runtime contract.
  The API production image compiles all command binaries in one Go build
  invocation so a cold bootstrap does not repeat dependency compilation or
  create avoidable intermediate layers on constrained developer machines.
  Browser acceptance invokes its checked-in Node entrypoint directly after the
  pinned Playwright install so it cannot trigger a competing pnpm modules purge
  while the web container is serving from that workspace.
- Lefthook and GitHub Actions run formatting, tests, generation drift checks,
  TypeScript checks, and builds.
- The generated mobile example uses the Expo-SDK-compatible, exactly pinned
  cryptographic UUID package for idempotency keys; every imported native module
  must be declared explicitly so a fresh bootstrap type-checks and builds.
- Rate limits, PostgreSQL pool sizes and lifetimes, account lifecycle, and
  observability exporters are environment-backed and validated.
- Liveness is process-only; readiness verifies migrations and required security
  dependencies. Structured access logs, metrics, traces, correlation IDs, and
  typed domain probes are injectable and fan-out capable. Optional validated
  OTLP/HTTP exporters emit real W3C-context traces and metrics.
- Authorization outbox retries have a configurable maximum-attempt dead-letter
  policy, cursor-paginated affected-owner visibility, atomically preclaimed
  audited replay, and metrics.
- The web app has a pinned non-root production image. Its build copies every
  workspace package imported by the web application, including `client-core`.
  Hardened Images run the generated API and web workloads as their catalog
  non-root UID. The Node builder installs the exactly pinned pnpm release with
  npm because the hardened Node image intentionally does not include Corepack,
  and writes deploy artifacts only beneath its unprivileged application
  workspace.
  Generated CI builds and scans API and web images and includes an
  immutable-action release workflow.
- SvelteKit owns nonce-based Content Security Policy generation. The server hook
  may adjust only `connect-src` for validated runtime API and OIDC origins so the
  production app hydrates without allowing inline script execution.
- Installation docs pin a concrete make-app release and never use `@latest`.

## Security guarantees

- Tokens validate signature, issuer, audience, and expiry through OIDC discovery.
- Generated authenticator tests cryptographically exercise valid tokens plus
  invalid signatures, wrong issuers, wrong audiences, expiry, and empty subjects.
- Routes do not trust identity headers supplied by callers.
- Resource reads require a SpiceDB permission check and hide inaccessible IDs.
- Audit history is scoped to events performed by the authenticated user or
  events affecting resources they own; callers cannot enumerate another user's
  activity through the audit API.
- Repository list operations require the authenticated owner ID.
- Local development uses Dex; production settings come from environment variables.

## Definition of ready

A release is not ready unless an empty-directory generation test proves all of
the following without manual source edits:

- generator unit tests and deterministic snapshot/structure checks pass;
- generated Go formatting, static analysis, unit tests, race tests, and builds pass;
- generated TypeScript checks, tests, OpenAPI drift checks, and production builds pass;
- the generated Expo application exports production bundles for both supported
  native targets, iOS and Android, without silently introducing a web target;
- pinned Compose configuration starts healthy PostgreSQL, SpiceDB, Dex, API, and web services;
- a real OIDC authorization-code-with-PKCE flow provisions exactly one local user;
- valid access, missing token, malformed token, invalid signature, wrong issuer,
  wrong audience, expired token, and concurrent first-login behavior are tested;
- owner creation, list, detail, update, and delete work through the public API;
- successful reads and writes, user provisioning, and denied cross-user
  attempts appear in immutable, cursor-paginated audit history without leaking
  another user's unrelated activity;
- pagination proves default and maximum limits, complete gap-free traversal,
  stable continuation under inserts, and rejection of malformed cursors;
- a second user cannot discover, read, change, delete, or forge ownership of the
  first user's resource, and denials do not reveal resource existence;
- SpiceDB unavailability fails closed, and interrupted relationship writes recover
  from a durable transactional outbox without orphaning or granting resources;
- concurrent outbox workers use owned expiring leases, cannot complete one
  another's claims, and recover an abandoned claim after its lease expires;
- every external authorization write is serialized per resource by a durable
  PostgreSQL lock held through the SpiceDB call, so an expired or delayed worker
  cannot reorder TOUCH and DELETE operations or resurrect access;
- PostgreSQL data, cursor-paginated audit history, and SpiceDB relationships survive
  a full generated-stack restart, including reauthentication after the local
  provider restarts; restart verification must traverse audit pages rather than
  assuming the newest event appears on the default page;
- example and blank live acceptance prove that an authorization dependency outage
  makes readiness unavailable while process liveness remains successful;
- migrations apply to an empty database and upgrade from the prior released schema;
- generated hooks are installed and CI runs the same authoritative verification command;
- dependencies, actions, tools, and runtime images are immutable and lockfiles are generated;
- Go-based release tools live in a dedicated checked-in module so their complete
  transitive graph is pinned, age-gated, and reviewed like application dependencies;
- every third-party CI action is selected by a reviewed immutable commit SHA;
- maintained first-party GitHub actions use releases whose action runtime is
  Node 24 or newer so generated CI does not depend on deprecated runner runtimes,
  and generated release workflows use the maintained unified attestation action
  rather than deprecated compatibility wrappers. Registry attestations disable
  organization-only artifact storage records so personal repositories can release;
- CI installs JavaScript dependencies from the generated frozen lockfile;
- the installed pre-push hook and CI invoke the same `make verify` release gate;
  pre-commit retains the documented fast, change-aware gate;
- the generator and every generated repository fail closed when an npm package
  or Go module in the resolved graph is less than fourteen days old;
- an age exception requires an exact ecosystem/name/version entry with a reason
  and compensating verification in `dependency-age-allowlist.json` plus a
  corresponding specification update;
- the generator loads reviewed dependency-age exceptions from its embedded
  template as well as its root, so the template is subject to the gate it installs;
- dependency metadata requests use bounded connection and total timeouts so the
  supply-chain gate fails closed instead of hanging a hook or CI job;
- web and mobile clients complete sign-in, refresh, `/v1/me`, authorized resource
  access, expiry handling, and sign-out through their adapters.
- browser acceptance drives the generated web client itself through Dex PKCE,
  callback exchange, application-session creation, and authenticated rendering;
  the local provider explicitly permits that public client's web origin.
- live acceptance rotates an application session, proves the old credential is
  immediately rejected, and continues with the replacement credential.
- PostgreSQL acceptance proves a rotation cannot extend a session beyond its
  original absolute family deadline.
- web and mobile select a supported device/browser locale, fall back safely to
  English, render translated copy and interpolation/plurals, and pass the
  untranslated-copy and catalog-parity structural gate;
- interactive API docs complete OIDC PKCE and successfully invoke `/v1/me` plus
  a protected resource operation without a client secret.
- the documentation token relay accepts the pinned Scalar public-client request
  shape, discards caller-supplied client and redirect identities, and constructs
  only a fixed authorization-code PKCE exchange for the configured docs client.

The live acceptance harness must run on every generator release. A skipped boundary
test is a release failure unless a reviewed specification records the temporary
exception, owner, risk, and removal date.
Generator CI exercises both the removable example application and a separately
generated `--without-example` application through frozen bootstrap, static
checks, client builds, Compose startup, OIDC, audited `/v1/me`, database-role
boundaries, and liveness/readiness behavior.
The PostgreSQL acceptance invocation runs the complete persistence-adapter test
package with a real DSN; name-based filtering must not silently omit a new
security boundary test.
Generator acceptance adds a real domain before bootstrap and runs every generated
domain repository's PostgreSQL integration test, including atomic create/delete
outbox, idempotency, timestamp, and audit guarantees, using the restricted API
runtime database credential rather than the migration owner.
The migration proof explicitly applies the prior release's complete migration
set and ledger state before invoking the current migrator; a hand-built baseline
that skips intervening migrations is not an upgrade test.
Generated CI also runs the live Compose acceptance harness, including an actual
pinned Scalar browser session that clicks Authorize, completes Dex login, and
uses Try It for `/v1/me` and a protected resource list. Playwright is an exact,
age-gated development dependency; that reviewed package fixes the downloaded
Chromium revision rather than resolving a floating browser release.
The browser harness tolerates only a bounded delay between a successful OIDC
token response and Scalar applying that credential to Try It requests; it retries
the real UI interaction and still fails if an authenticated request never occurs.
The same browser acceptance opens the generated web client with a regional
Spanish browser locale and proves base-locale negotiation, the document language,
and translated UI copy at the real rendering boundary.
Generated release planning consumes the successful exact-SHA CI evidence and
does not rerun the live acceptance gate for the same commit.
The publish job scans both locally built candidate container images for high and
critical vulnerabilities with a full-SHA-pinned scanner before pushing them.
It then captures and attests their registry digests. Immediately before digest
promotion and Git tagging, it re-fetches `main` and rejects a stale source SHA.
The live harness starts and waits for the generated web service before these
checks; API-only readiness is not sufficient for frontend acceptance.
Each live acceptance invocation uses a unique Compose project name and removes
its project before startup and on exit, so named volumes and containers from an
interrupted run cannot contaminate migration or persistence proof. Because the
browser-visible local OIDC issuer intentionally uses fixed localhost ports,
acceptance also holds a bounded host lock; simultaneous invocations serialize
instead of colliding or connecting to another run's services.
The generated harness stops the API while direct PostgreSQL adapter integration
tests run, preventing the live authorization worker from racing those tests for
outbox rows, then restarts the API and proves readiness before HTTP acceptance.
Generated behavioral tests cover fallback, interpolation, plural forms,
locale-aware numbers/dates, and the Expo device-locale adapter; locale-dependent
SSR responses are required to emit `Vary: Accept-Language`.

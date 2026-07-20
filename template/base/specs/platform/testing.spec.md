# Testing and Delivery Specification

Tests use fakes rather than mocks and validate behavior through ports. Every auth
boundary covers missing, invalid, expired, wrong-issuer, wrong-audience,
cross-user, dependency-failure, and legitimate cases where applicable.
Audit tests prove atomic mutation/event commits, append-only database enforcement,
successful read and command coverage, denied-decision coverage, actor/owner
visibility, cross-user isolation, stable pagination, and persistence across a
complete stack restart. Persistence acceptance follows cursor pagination until it
finds the pre-restart audit event; it must not depend on a default page being large
enough for the accumulated acceptance history.
Platform audit adapter tests do not depend on an installed product domain or
create test-only product storage. The removable example owns its resource/audit
atomicity proof, and every added domain owns the corresponding repository proof.
Account lifecycle tests prove self-deactivation is configuration-gated, atomically
disables the account, revokes all of its sessions, writes audit history, and
rejects every old session and later OIDC exchange.

Pre-commit and CI run Go formatting and tests, structural checks, OpenAPI/client
drift checks, TypeScript checks, and production builds. Dependencies, CI actions,
toolchains, and images are pinned. Generated projects must pass checks immediately
after bootstrap without manual source edits.
Mobile verification names its evidence precisely. Fast checks run Expo Doctor,
Expo dependency compatibility, JavaScript bundle export, configuration/identifier
validation, and clean iOS/Android prebuild. Linux CI compiles an unsigned Android
debug application with Gradle. A macOS main/release/manual job compiles an unsigned
iOS simulator application with Xcode. Export success alone is never reported as
a native build. Session-state tests prove that transient network, 429, and 5xx
failures preserve a locally valid credential, while expiry, explicit 401, and
malformed secure storage remove it.
Shared web/mobile refresh adapter tests cover network loss, 401, 429, 503,
successful rotation followed by profile-read failure, credential retention or
disposal, bounded retry, and expiry enforcement.
Production mobile configuration tests reject missing, non-HTTPS, loopback, and
localhost endpoint values in the actual Expo bundle environment.
First-party GitHub actions are pinned by full commit SHA to maintained releases
whose action runtime is Node 24 or newer; generated CI must not emit deprecated
action-runtime warnings. Release attestations use the maintained unified GitHub
attestation action, not deprecated compatibility wrappers, and disable optional
organization-only artifact storage records for personal-repository compatibility.
Release tooling, including EAS CLI, is installed from the same frozen pnpm
lockfile and is included in dependency-age and vulnerability gates.
The locked EAS executable retains the mobile app as its working directory.
CocoaPods is Bundler-locked, Android CI selects an exact JDK patch, and signed
EAS profiles select reviewed named Expo SDK 55 build images.
Hosted iOS compilation runs on the explicit GitHub `macos-26` image because
Expo SDK 55 requires Xcode 26 or newer.
Every locked Ruby gem participates in dependency-age and OSV vulnerability
checks. The normal generated check starts the real locked EAS executable.
Reviewed overrides raise vulnerable EAS transitive parsers and matchers to fixed
versions; the locked CLI must pass version startup, Expo Doctor, and audit after
any override change.
The initial locked EAS 21.0.2 toolchain has exact, reviewed age exceptions for
`eas-cli`, its matching `@expo/eas-build-job`, `@expo/eas-json`, `@expo/steps`,
`@expo/logger`, `@expo/turtle-spawn`, and resolved `multipasta`. Compensating
verification includes the frozen graph, fixed security overrides, clean audit,
CLI startup, Expo Doctor, prebuild, and hosted native compilation.
Migration acceptance applies the prior released migration set first, then runs
the current migrator and verifies preserved baseline data and new schema objects.
Every prior-release up and down migration is frozen by a checked-in SHA-256
inventory that normal tests enforce before the live PostgreSQL upgrade proof.
Each supported generated baseline variant selects a separate reviewed inventory.
PostgreSQL adapter acceptance also verifies that session rotation revokes the old
credential atomically and caps the replacement at the original family deadline.

The release gate runs Go vulnerability analysis and the package-manager audit
against the resolved dependency graph. CI has least-privilege permissions,
bounded runtime, and cancels superseded work. High or critical dependency
findings at any severity fail delivery rather than being silently accepted.
The API dependency graph pins `golang.org/x/net v0.55.0`, excluding the
GO-2026-5026-affected v0.53.0 release from generated modules and images.
Release image scans use the locally installed Grype 0.111.1 executable from a
checksum-verified immutable release archive. Scanner bootstrap must not execute a
mutable branch installer or delegate installation to the scan action.
CI runs the live Compose acceptance harness. A pinned Playwright/Chromium browser
must operate Scalar itself: authorize through Dex with PKCE, then send authenticated
Try It requests to `/v1/me` and a protected resource endpoint. Protocol-only
reconstruction is supporting evidence, not a substitute for this browser boundary.
The harness permits bounded UI retries while Scalar renders the Try It request
control and applies a successfully exchanged credential, but must fail if the
control never appears or Scalar never attaches the bearer credential.
Every harness invocation has a unique Compose project and performs project and
volume cleanup both before setup and on exit. Interrupted, stale, or concurrent
runs must not share migration state or persistence fixtures.
Harness assertions capture bounded multi-line producer output before applying
early-exit searches, so `pipefail` cannot relabel a successful match as a
nondeterministic upstream SIGPIPE failure.
Each added domain carries an adversarial HTTP composition test proving its
generated routes are registered, missing or invalid sessions return 401, and a
valid session receives 503 until an explicit authorization policy replaces the
fail-closed scaffold. The test exercises the real Huma route boundary rather
than substituting a direct service-only assertion.
It also proves an authentication dependency failure is not rewritten as an
invalid credential. Multi-domain generator tests include domain names whose
separator-stripping title forms would otherwise collide in Huma.
The browser also opens the web client with `es-ES`, verifies negotiation to the
supported `es` locale, checks the document language, and observes Spanish catalog
copy. Static catalog checks alone are not sufficient rendering evidence.
Shared runtime tests exercise supported and unsupported locale selection,
interpolation, singular and plural forms, and locale-aware number/date formatting.
The Expo adapter is tested with controlled device-locale readers for supported
regional and unsupported locales. These behavioral tests run in `make check`,
the pre-commit release gate, and CI.
Playwright is exact and age-gated. Its exact package version fixes the Chromium
revision used by acceptance; the browser is not independently represented as an
npm dependency and must not be described as independently age-gated.
Go release tools and their transitive dependencies are pinned in the dedicated
`tools` module, which is covered by the same dependency-age gate.
The generated web development service uses the digest-pinned hardened Node 24
builder and npm so its exact pnpm package-manager pin can start without an
unpinned global install. Its Compose environment enables pnpm's
non-interactive CI behavior so a stale or host-created modules directory cannot
block first startup waiting for a terminal prompt. The service runs with the
unprivileged `MAKE_APP_UID` and `MAKE_APP_GID` (both defaulting to 1000) so it
does not leave root-owned artifacts in the bind-mounted generated repository.
The supported Make and acceptance entrypoints populate those values from
`id -u` and `id -g`, including on CI hosts whose user is not UID 1000. Writable
temporary HOME, XDG cache, and npm prefix/cache locations make arbitrary numeric users
independent of `/etc/passwd`; release acceptance runs the image as 1001:1001.
The repository-local ignored pnpm store is shared by host bootstrap and the web
container. Live acceptance gives a cold container installation up to ten minutes
to become ready on a slow registry, then fails if the web boundary is unavailable.
Once Compose is live, browser acceptance runs through Node directly; it must not
ask pnpm to reconcile the bind-mounted dependency tree during the live test.
Browser acceptance drives the generated SvelteKit client through Dex sign-in,
the callback code exchange, application-session exchange, and authenticated
profile rendering. An unauthenticated localization-only render is insufficient.
Frontend session adapters validate exchanges before persistence, retry only
retryable failures, and preserve authenticated-offline presentation state. A
mobile orchestration test cold-starts with a valid stored credential while OIDC
discovery and the API are unavailable, and proves restoration completes in the
authenticated-offline state without deleting the credential. Default and blank
mobile clients invoke that restoration independently of provider discovery.
Post-exchange orchestration proves a profile 401 deletes the newly exchanged
credential while a profile 503 retains it and enters authenticated-offline.
Controlled loopback transport tests prove generated session exchange, refresh,
profile validation, and revocation request paths, bodies, credential ownership,
and status propagation without replacing the network transport. The structural
gate uses a fail-closed AST/import allowlist to reject direct, computed, aliased,
beacon, and dynamically imported application transports in both clients while
permitting provider traffic only through the exact OIDC adapters. Fixtures cover
same-directory `.mjs` and `.cjs` transports, unresolved or unsupported `$lib`
aliases, bare-window aliases and destructuring, and relative-import escapes into unapproved
shared helper roots.
Generator unit fixtures that exercise dependency-free structural rules block
package installation and run with no `node_modules`; bootstrapped default and
blank acceptance exercise the real pinned AST parser.
The web runtime configuration tests reject absent and unsafe production API,
issuer, and client settings. Container acceptance proves the production image
fails before serving without them while local Compose explicitly selects the
development contract.
PostgreSQL adapter integration tests run while the API service is stopped so its
authorization outbox worker cannot consume test fixtures concurrently. The
harness restarts the API and re-establishes readiness before exercising live HTTP
boundaries.
Both example and blank live harnesses stop SpiceDB and assert that `/readyz`
fails closed with 503 while `/livez` remains a dependency-independent 204.

npm packages and Go modules must be at least fourteen days old. The age gate
fails closed when registry metadata cannot be retrieved or parsed. A reviewed
exception must pin one exact ecosystem, name, and version in
`dependency-age-allowlist.json`, state why it is required, name compensating
verification, and be recorded in this specification. Lowering the global age
threshold is not an acceptable shortcut.
Metadata requests have bounded connection and total timeouts; unavailable
registries fail the gate closed instead of hanging local hooks or CI.
Broad transitive ranges are constrained when necessary to keep resolution
reviewable and age-compliant. The React Native/Jest 29 schema edge pins compatible
`@sinclair/typebox 0.27.8` instead of accepting newly published artifacts from
its broad range.

Reviewed baseline exceptions align native OIDC with the pinned Expo SDK 55
runtime: `expo-auth-session 55.0.17`, `expo-web-browser 55.0.18`,
`expo-secure-store 55.0.16`, `expo-application 55.0.17`, `expo-crypto 55.0.17`,
and `expo-linking 55.0.16`. Their compensating checks are exact pins, resolved
lockfile review, a clean vulnerability audit, Expo compatibility validation,
mobile type checking, and live OIDC/API acceptance as applicable.

The structural gate rejects oversized handwritten Go files, mocks, ad hoc print
calls, direct SQL helpers, environment reads outside configuration/bootstrap,
floating CI action references, and container images without immutable digests.
Generated delivery tests also enforce the reviewed Red Hat Hardened Images
boundary: Go and Node build stages, API and web runtimes, and PostgreSQL use
immutable HI references, while unsupported SpiceDB and Dex components remain
version-and-digest-pinned upstream images. Live acceptance proves that the
hardened PostgreSQL entrypoint, non-root Node runtime, and static Go runtime
preserve the generated application contract.
Generated Go files carrying the standard `Code generated ... DO NOT EDIT` marker
are exempt only from the line-length rule.
The structural gate also rejects literal user-facing text and translatable
attributes in Svelte and JSX/TSX. It validates every locale as non-empty and
key-complete against English, verifies matching interpolation parameters and
plural pairs, and runs before dependency installation as part of the normal gate.
Object property keys used only as protocol syntax, including HTTP header names,
are not copy; the gate must distinguish them from nested literal JSX expressions.

Routine pre-commit checks are change-aware. They always run structural,
formatting, contract-drift, and focused tests, and must run dependency-age and
resolved-graph security gates whenever a dependency manifest, lockfile, action,
tool module, or container definition changes. Full race, production-build, and
live acceptance gates remain mandatory before push and release. Release
publication may reuse successful CI evidence only when it is tied to the exact
commit SHA and fails closed if that evidence is absent.
Pre-commit and pre-push retain the hook-owned index for direct caller-repository
comparisons, including staged-change classification. Recursive check, generation,
dependency, verification, and acceptance make processes clear inherited Git
directory, worktree, index, and prefix variables so nested fixture repositories
cannot read or mutate the caller's hook index.
Tests that create temporary Git repositories clear inherited repository-local
Git environment variables, disable fixture hooks, and prove they leave the
caller's staged name-status unchanged, including under pre-commit alternate-index
execution.

Static Compose tests reject host networking and prove `.env` loading. Live
acceptance exercises bridge-network service discovery and the OIDC backchannel
without weakening public issuer validation. Acceptance cleanup removes its
project-scoped containers, volumes, and locally built images so repeated clean
runs do not exhaust a development or CI host.

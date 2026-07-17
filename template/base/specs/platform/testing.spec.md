# Testing and Delivery Specification

Tests use fakes rather than mocks and validate behavior through ports. Every auth
boundary covers missing, invalid, expired, wrong-issuer, wrong-audience,
cross-user, dependency-failure, and legitimate cases where applicable.

Pre-commit and CI run Go formatting and tests, structural checks, OpenAPI/client
drift checks, TypeScript checks, and production builds. Dependencies, CI actions,
toolchains, and images are pinned. Generated projects must pass checks immediately
after bootstrap without manual source edits.
Migration acceptance applies the prior released migration set first, then runs
the current migrator and verifies preserved baseline data and new schema objects.

The release gate runs Go vulnerability analysis and the package-manager audit
against the resolved dependency graph. CI has least-privilege permissions,
bounded runtime, and cancels superseded work. High or critical dependency
findings at any severity fail delivery rather than being silently accepted.
CI runs the live Compose acceptance harness. A pinned Playwright/Chromium browser
must operate Scalar itself: authorize through Dex with PKCE, then send authenticated
Try It requests to `/v1/me` and a protected resource endpoint. Protocol-only
reconstruction is supporting evidence, not a substitute for this browser boundary.
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
The generated web development service uses digest-pinned Node 24 Alpine with
the reviewed bundled Corepack version so its exact pnpm package-manager pin can
start without an unpinned global install. Its Compose environment enables pnpm's
non-interactive CI behavior so a stale or host-created modules directory cannot
block first startup waiting for a terminal prompt. The service runs with the
unprivileged `MAKE_APP_UID` and `MAKE_APP_GID` (both defaulting to 1000) so it
does not leave root-owned artifacts in the bind-mounted generated repository.
The supported Make and acceptance entrypoints populate those values from
`id -u` and `id -g`, including on CI hosts whose user is not UID 1000. Writable
temporary HOME, XDG cache, and Corepack locations make arbitrary numeric users
independent of `/etc/passwd`; release acceptance runs the image as 1001:1001.
The repository-local ignored pnpm store is shared by host bootstrap and the web
container. Live acceptance gives a cold container installation up to ten minutes
to become ready on a slow registry, then fails if the web boundary is unavailable.
Once Compose is live, browser acceptance runs through Node directly; it must not
ask pnpm to reconcile the bind-mounted dependency tree during the live test.

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
runtime: `expo-auth-session 55.0.17`, `expo-web-browser 55.0.17`,
`expo-secure-store 55.0.15`, `expo-application 55.0.16`, `expo-crypto 55.0.16`,
and `expo-linking 55.0.15`. Their compensating checks are exact pins, resolved
lockfile review, a clean vulnerability audit, Expo compatibility validation,
mobile type checking, and live OIDC/API acceptance as applicable.

The structural gate rejects oversized handwritten Go files, mocks, ad hoc print
calls, direct SQL helpers, environment reads outside configuration/bootstrap,
floating CI action references, and container images without immutable digests.
Generated Go files carrying the standard `Code generated ... DO NOT EDIT` marker
are exempt only from the line-length rule.
The structural gate also rejects literal user-facing text and translatable
attributes in Svelte and JSX/TSX. It validates every locale as non-empty and
key-complete against English, verifies matching interpolation parameters and
plural pairs, and runs before dependency installation as part of the normal gate.

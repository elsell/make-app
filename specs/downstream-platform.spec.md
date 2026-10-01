# Downstream platform contributions

## Purpose and scope

Make App must incorporate reusable platform improvements from HourPaths and
Stuff Stash without importing their product domains, private hosts, credentials,
tenancy assumptions, or deployment policies. Existing generated applications
must retain their established authentication, authorization and audit boundaries.

## Baseline safeguards

- Generated repositories must inventory application environment reads across Go,
  JavaScript/TypeScript/Svelte, Docker, Compose and example configuration. Only
  the generated prefix and documented tool variables are allowed. Computed reads
  require an adjacent, single-use review annotation.
- Contract verification must generate OpenAPI and TypeScript in temporary storage,
  compare both artifacts, fail on drift, and leave tracked and untracked client
  files unchanged on success or failure.
- CI must classify a complete diff once. Unknown paths, malformed input and
  unavailable history must select all expensive gates. Verification always runs;
  stable required jobs must fail if classification fails. Trusted main pushes and
  manual runs must run all gates, providing actual platform evidence for release.
  Existing Docker cache verification must be preserved.
- Android CI must install and verify the reviewed SDK, NDK, CMake and Java set
  before Gradle. Tests must reject a missing required component.
- Production mobile URL validation must reject credentials, query/fragment,
  loopback, unspecified, private IPv4/IPv6 and IPv4-mapped local IPv6 literals.
  Private deployments require a separately reviewed policy; no silent bypass.
- Architecture guards must reject domain infrastructure imports and transport
  concerns in DTOs; explicitly reviewed infrastructure SQL remains separate.

## Reusable client behavior

Client core must provide dependency-free coordination primitives with behavioral
coverage: blocking/draining in-flight mutations before sign-out; rejecting late
results after a session or account transition; coalescing refresh signals without
concurrent requests; and disposal that suppresses subsequent effects. Browser
focus/visibility and inter-tab signals belong behind an injected adapter, with
best-effort operation when browser channels are unavailable. Signals are hints to
refetch authorized data, never trusted content or permission grants.

Mobile helpers must isolate pull-gesture presentation from background loading,
clear it on navigation, ignore obsolete completion, and present stable native
header options while invoking current committed handlers. Removed, disabled and
unmounted actions must not execute. Helpers must not impose product navigation.

## Optional capabilities

Store delivery, self-host packaging, notifications and media must be explicitly
selected and remain absent from default runtime wiring. Optional components must
ship executable behavior and tests, not only interface placeholders. Product
routes and authorization policies must remain application-owned and fail closed.

### Store delivery

Provide a manually enabled protected TestFlight workflow using a published tag,
resolved immutable commit, successful CI jobs for that commit, production-only
configuration, ephemeral signing material, locked native dependencies, verified
archive identity/version and cleanup on failure. Recovery must rebuild an existing
published tag using a fresh build number without retagging. Release notes must
target and verify the exact app/version/build. Profile inspection and repair must
validate app/team/certificate identity and must not silently rotate credentials.
Apple entitlements, encryption declarations and notification capability must be
explicit application policy rather than copied Stuff Stash assumptions.

### Self-host packaging

Provide an opt-in recipe containing digest-pinned API/web references, Compose,
version and checksums. Configuration and preflight must reject defaults that are
unsafe in production. Separate migration, runtime and authorization-proxy
credentials must be preserved. Packaging must not include secrets or build output.

### Notifications and media

Notifications must expose a provider-neutral send/receipt/failure contract with
injected transport and clock, bounded retries, invalid-device handling and safe
errors. APNs, FCM and Expo integrations must be optional adapters. Delivery policy,
recipient authorization, persistence/outbox ownership and message copy stay in the
application. Tests must cover provider rejection and transient failures.

Media must expose owner-neutral blob storage, bounded direct upload, image
processing and deferred deletion primitives behind injected ports. Filesystem and
S3-compatible adapters must validate keys and prevent traversal; upload completion
must validate identity, expiry, size and content before publication. Product
attachment ownership and authorized endpoints are not generated implicitly.

## Acceptance

Each contribution must include focused behavioral tests and generated-repository
coverage. Both example and blank generation must remain valid. Native compilation,
live providers, signed store publication and deployed self-host acceptance must
be reported separately from local source/unit-test evidence. Contributions may be
reviewed as atomic commits or separate PRs; all requested capabilities remain in
scope until delivered or a concrete external blocker is reported.

## Feature installation command

`make-app feature add NAME --dir PATH` must support notifications, media,
selfhost and testflight. It must validate the generated manifest, refuse unknown
names, symlinks, duplicate installations and existing target files, stage and
format files before installation, and roll back newly installed files on failure.
The manifest records selected features without changing its schema version;
older generator versions must not be used to rewrite a feature-enabled manifest.
Features add files only; application composition remains explicit.

Media is an isolated optional Go module (`modules/media`) so its pinned MinIO
v7.2.0 dependency graph does not affect default applications. MinIO is justified
for reviewed S3 signing, bounded POST policies and compatible storage rather than
implementing a custom cryptographic protocol. Application composition must select
its ownership model and add module/workspace references explicitly. The generated
check target runs installed feature checks. Self-host packaging provides a secure
recipe using externally operated TLS PostgreSQL, SpiceDB and OIDC services; it
must not import development service accounts as a production topology.

Direct uploads must target a quarantine key distinct from the immutable published
key. Completion reads and validates quarantine bytes, then uses exclusive creation
to publish the validated content. Replaying a still-valid upload policy must never
change published bytes. Repeated completion may fail closed; application-owned
outboxes clean up abandoned quarantine/published objects after transaction failure.

Media content inspection is injected through a port. Release publication attaches
the optional self-host bundle and checksum using the attested image digests.
Refresh coalescing must also serialize synchronously reentrant requests.

Generated security checks must scan installed optional Go modules as well as the
API, so an isolated dependency graph cannot silently escape vulnerability checks.

Android preflight accepts an explicit `--install` mode for hosted CI. It resolves
sdkmanager from the configured SDK when the runner omits it from PATH, installs
the reviewed exact package versions, then verifies all required tools.

The generator and optional media module require Go 1.25.13 to include the fix
for GO-2026-5972 (published August 13, 2026; older than the fourteen-day gate).
Optional module source files remain template assets, not packages in the
generator's own module; generated-feature CI tests and scans their actual graph.
Existing application/container toolchain upgrades remain an explicit deployment
composition change when integrating media.

# Development workflow

Run `make-app doctor`, then copy `.env.example` to `.env` and run
`make bootstrap`. The bootstrap installs the exact workspace dependencies,
regenerates the OpenAPI client, and runs the initial checks.

`make dev` starts PostgreSQL, SpiceDB, its capability proxy, migrations, schema
application, and Dex in Docker. It runs the API and SvelteKit development server
on the host. Go and migration changes restart the API; SvelteKit supplies its
normal hot-module reload. Stop the foreground command with Ctrl-C. Stateful
containers remain available for the next run; use `make compose-down` to stop
them.

Common focused commands:

- `make api`, `make web`, or `make mobile` runs one application.
- `make logs` follows container logs.
- `make db-shell` opens the migration-role PostgreSQL shell for local diagnosis.
- `make migrate` reapplies the one-shot migration job.
- `make seed` creates three example records through OIDC and the public API.
- `make reset RESET=1` intentionally removes local containers and volumes.
- `make compose-up` builds and runs production-like API and web images.

The pre-commit hook runs focused checks and performs dependency-age and
vulnerability work when dependency inputs changed. Pre-push and CI run the full
verification and live acceptance boundaries.

## Shared client coordination

Client core exports `createAsyncMutationBarrier` to stop new writes and drain
already admitted mutations before sign-out. Release each lease in `finally`.
Invalidate a `createSessionScope` whenever the account or session changes; capture
its lease before asynchronous reads and check `current()` before committing data.
This avoids restoring a prior account's data after a late response.

`createNotificationRefreshLatch` coalesces refresh signals and supplies a validity
callback to suppress stale/disposed results. Check it immediately before committing
response data. A failed refresh still drains a queued refresh; if the last refresh
fails, callers receive that failure. It does not queue offline mutations.

For web tabs, compose `openNotificationConvergenceBrowser` with the reviewed
`browserEvents` adapter and an application-specific channel name. Messages are
untrusted hints to refetch authorized data. Include the current account/session
identity when deciding whether to act on a hint. Browser focus and visibility
remain fallbacks if BroadcastChannel is unavailable. Close the adapter on teardown.

`python3 scripts/check-generated-contracts.py` verifies contracts in temporary
storage. It never overwrites your local changes; use `make generate` explicitly
when you intend to update the generated files.

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

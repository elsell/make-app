# __APP_NAME__

A generated Go, PostgreSQL, SpiceDB, OIDC, SvelteKit, and Expo application.
Web and mobile share typed English and Spanish locale catalogs in
`packages/i18n`; the structural gate prevents untranslated UI copy and incomplete
catalog updates.

Audit history is a mandatory platform primitive. User provisioning and every
authenticated domain read, list, state change, and denied authorization decision
produce append-only events. Successful mutations commit with their audit event in
the same PostgreSQL transaction. A signed, cursor-paginated `GET /v1/audit-events`
endpoint returns only events visible to the current actor or resource owner.
OIDC tokens are exchanged once for rotating opaque application sessions. Set
`__ENV_PREFIX___SESSION_TTL_MINUTES` for each credential and
`__ENV_PREFIX___SESSION_ABSOLUTE_TTL_MINUTES` for the non-extendable session-family
deadline. Once that deadline is reached, the user signs in with OIDC again. Set
`__ENV_PREFIX___ACCOUNT_PROVISIONING_MODE=existing` with a normalized
`__ENV_PREFIX___ACCOUNT_INVITED_EMAILS` list for invitation-only signup. Optional
OTLP/HTTP trace and metric export is configured with the `OTEL_*` variables in
`.env.example`.

## Start locally

Install Go 1.25.12, Node 22.17.0, pnpm 11.0.7, Python 3, Make, Git, and Docker
Compose. Run `make-app doctor` before generation to check them together.

```sh
make bootstrap
make dev
```

Dex is available at `http://localhost:5556/dex`, the API at
`http://localhost:8080`, and the web app at `http://localhost:5173`.
The local test account is `developer@example.com` with password `password`.

`make dev` runs stateful dependencies in portable bridge-network containers and
the API/web processes on the host with reload-on-change. Use `make compose-up`
to exercise the production images instead. `.env` configures host development
and policy settings in Compose; Compose overrides only its internal service
addresses.

Useful commands include `make mobile`, `make logs`, `make db-shell`,
`make migrate`, `make seed`, and the guarded `make reset RESET=1`. See
[`docs/development.md`](docs/development.md) and [`docs/mobile.md`](docs/mobile.md).

Run `make-app domain add habit` to scaffold a dedicated table, migration,
repository, DTOs, mapper, and routes for another owner-authorized domain. You
still write its application service and sharing policy.
Use `--fields 'title:string,active:bool,target:int,due_at:time'` and, when
needed, `--plural entries`. The command prints the remaining composition steps;
[`docs/domains.md`](docs/domains.md) walks through them. Generate without the
demonstration slice using `--without-example`, or later run
`make-app example remove`.

Before deploying, follow [`docs/oidc.md`](docs/oidc.md) and
[`docs/production.md`](docs/production.md).

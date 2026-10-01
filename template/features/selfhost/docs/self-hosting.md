# Self-host release bundle

This production recipe runs the API, web frontend and restricted authorization
proxy on one Docker host. Bring a TLS PostgreSQL service, TLS SpiceDB service,
OIDC provider and HTTPS reverse proxy. These external services own their durable
storage and backups. It deliberately does not ship development database users or
Dex accounts. See `docs/production.md` in the source repository for role grants.

Create separate PostgreSQL migration/runtime roles and separate SpiceDB schema/
runtime secrets. Copy `deploy/selfhost/environment.example` to a private `.env`
and fill every value. Use independent randomly generated secrets of at least
32 characters. Keep `.env` outside version control and chmod it to 600.

```sh
set -a; . ./.env; set +a
python3 scripts/selfhost-preflight.py
docker compose --env-file .env -f deploy/selfhost/compose.yaml config --quiet
docker compose --env-file .env -f deploy/selfhost/compose.yaml up -d
```

The host reverse proxy forwards HTTPS to loopback ports 8080 and 5173. The
capability proxy is reachable only inside the private Compose network; its
connection to external SpiceDB uses TLS. Do not expose the proxy port. The API
never receives the schema credential or migration DSN.

Once installed, the normal release workflow attaches the bundle and checksum
using the scanned, attested image digests. To package a reviewed release locally, run `python3 scripts/build-selfhost-release.py`
with `--version vX.Y.Z`, `--api-image registry/app@sha256:...` and
`--web-image registry/web@sha256:...`. The archive and SHA-256 file are written to
`release-dist`. Verify the checksum before extracting. Credentials are never
packaged. Test migrations, readiness, sign-in and backup restore in staging before
production; reverting an image does not automatically roll back schema changes.

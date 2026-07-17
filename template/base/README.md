# __APP_NAME__

A generated Go, PostgreSQL, SpiceDB, OIDC, SvelteKit, and Expo application.
Web and mobile share typed English and Spanish locale catalogs in
`packages/i18n`; the structural gate prevents untranslated UI copy and incomplete
catalog updates.

## Start locally

```sh
cp .env.example .env
pnpm install
make generate
make compose-up
```

Dex is available at `http://localhost:5556/dex`, the API at
`http://localhost:8080`, and the web app at `http://localhost:5173`.
The local test account is `developer@example.com` with password `password`.

Run `make-app domain add habit` to scaffold another owner-authorized domain.

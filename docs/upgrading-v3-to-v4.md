# Upgrade a generated project from schema 3 to schema 4

Schema 4 adds generated application-service dependency injection and automatic
Huma route registration. Those files are composition and security boundaries,
so `make-app` deliberately does not rewrite a schema-3 repository in place.

Do not change `schemaVersion` by hand. That makes the manifest claim wiring that
the repository does not have.

Use this side-by-side procedure:

1. Commit the schema-3 repository, record the deployed release, and back up its
   PostgreSQL database. Stop if either the working tree or migration state cannot
   be reproduced.
2. Save `.make-app.json` and the complete, numerically ordered set of migration
   files. The
   manifest is the source for the app name, module, bundle prefix, domain names,
   explicit plurals, and field specifications. The migration filenames are the
   source for domain migration order.
3. With a schema-4 `make-app`, generate a new sibling repository using the same
   name, module, and bundle prefix. Pass `--without-example` exactly when the
   schema-3 manifest does not contain the `example` domain.
4. Reconstruct every migration after the generated baseline in numeric order.
   For an existing generated domain migration, run `make-app domain add` with
   that domain's manifest plural and fields. For an intervening product
   migration, copy its matching `.up.sql` and `.down.sql` files into the sibling
   before adding the next domain. Update `LatestVersion` in
   `apps/api/internal/adapters/dbmigrations/migrations.go` to the copied version;
   the next `domain add` will then take the next unused filename. Never leave a
   gap or reorder entries.
5. Generated domain migration filenames use the normalized singular domain plus
   `s`, such as `000016_create_categorys`, even when the route has an irregular
   explicit plural such as `categories`. Each regenerated filename and version
   must match its schema-3 counterpart. Compare both directions byte-for-byte;
   keep the schema-3 content whenever product development changed it. If a name
   or version differs, stop and diagnose the sequence instead of renumbering an
   applied migration.
6. After reconstructing the sequence, verify `LatestVersion` equals the highest
   migration filename and run the migration package tests against both an empty
   database and a restored backup. Never edit an already applied migration.
7. Port product specs first, followed by domain/application behavior, adapters,
   client UI, translations, tests, and deployment configuration. Keep the
   schema-4 generated composition files and generated fail-closed boundary tests.
   Resolve changes deliberately; do not copy the old repository over the new
   scaffold as a whole.
8. Implement each generated domain's authorization policy before exposing it.
   Until then, authenticated calls correctly return `503` and cannot reach its
   repository. Preserve the generated unauthenticated, malformed-credential,
   dependency-failure, and policy-not-configured cases.
9. Point a disposable environment at a restored database backup and run
   `make bootstrap`, `make verify`, and `make acceptance`. Confirm migrations do
   not attempt to recreate or skip domain tables and that the generated OpenAPI
   document contains every domain route.
10. Deploy the schema-4 repository as a normal application release. Keep the old
   release and database backup available for rollback until production health,
   authorization reconciliation, and audit delivery are verified.

This is intentionally a repository migration, not a manifest edit. Generated
projects own their code, and arbitrary product changes cannot be merged safely
by a generic source rewriter.

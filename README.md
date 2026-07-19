# make-app

Generate a working Go, PostgreSQL, SpiceDB, OIDC, web, and Expo monorepo.

<!-- make-app:release-version:start -->
```sh
go install github.com/elsell/make-app@v0.4.2
make-app doctor
make-app new "Habit Kit" --module github.com/you/habit-kit
cd habit-kit
make bootstrap
make-app domain add habit
```
<!-- make-app:release-version:end -->

To adopt an existing spec-first repository without replacing its Git history:

```sh
cd hour-paths
make-app init "Hour Paths" --module github.com/you/hour-paths --without-example
make bootstrap
```

`init` accepts only an existing repository containing guidance, specs, docs, and
related metadata. It merges marked guidance, preserves product specs, and refuses
conflicts before mutation.

Use `--without-example` for a blank product surface. Domain scaffolds accept an
explicit plural and typed fields, for example:

```sh
make-app domain add category --plural categories --fields 'title:string,active:bool,target:int,due_at:time'
```

Generated repositories start on `main`, include change-aware pre-commit and full
pre-push gates, and carry a template compatibility manifest. `make dev` provides
the hot-reload workflow; `make-app example remove` removes the demonstration
slice later through a forward migration.

The generated application owns all of its code. See `specs/generator.spec.md` for
the contract and generated security model. Audit history is included from the
first migration: mutations and their events are atomic, records are append-only,
and users get an isolated, cursor-paginated audit API.

Mobile delivery distinguishes Expo export from native compilation. Generated
projects include Expo Doctor and dependency checks, clean prebuild, Android
Gradle and iOS simulator CI, `expo-dev-client`, guarded EAS profiles, release
metadata and assets, and a shared session state machine that keeps valid
credentials during transient outages. Cold-launch restoration reads secure local
storage independently of OIDC discovery, so a valid session can enter the
authenticated-offline state without provider or API connectivity.

Generated API, web, and PostgreSQL containers use digest-pinned Red Hat Hardened
Images where the catalog has a compatible component. The production web image
fails closed when explicit secure API and OIDC configuration is missing; local
Compose selects the reviewed development defaults explicitly.

`make-app` is an alpha-quality `v0.x` project licensed under Apache-2.0. See the
[compatibility table](docs/compatibility.md) and [security policy](SECURITY.md).

Projects created with template schema 3 must follow the
[v3-to-v4 repository upgrade procedure](docs/upgrading-v3-to-v4.md) before using
schema-4 mutation commands; changing only `.make-app.json` is not supported.

# make-app

Generate a working Go, PostgreSQL, SpiceDB, OIDC, web, and Expo monorepo.

```sh
go install github.com/elsell/make-app@v0.1.0
make-app doctor
make-app new "Habit Kit" --module github.com/you/habit-kit
cd habit-kit
make bootstrap
make-app domain add habit
```

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

Projects created with template schema 3 must follow the
[v3-to-v4 repository upgrade procedure](docs/upgrading-v3-to-v4.md) before using
schema-4 mutation commands; changing only `.make-app.json` is not supported.

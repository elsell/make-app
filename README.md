# make-app

Generate a working Go, PostgreSQL, SpiceDB, OIDC, web, and Expo monorepo.

```sh
go install github.com/elsell/make-app@v0.1.0
make-app new "Habit Kit" --module github.com/you/habit-kit
cd habit-kit
make bootstrap
make-app domain add habit
```

The generated application owns all of its code. See `specs/generator.spec.md` for
the contract and generated security model. Audit history is included from the
first migration: mutations and their events are atomic, records are append-only,
and users get an isolated, cursor-paginated audit API.

# Generator and template compatibility

This repository was generated from template schema 4. The generator version and
schema are recorded in `.make-app.json`; include both in upgrade and support
requests.

Generated applications own their source. Installing a newer `make-app` binary
does not rewrite an application. Mutation commands fail closed when the project's
schema is unsupported and link to an explicit upgrade procedure when one exists.
Do not change the schema number without applying the complete documented upgrade.

The generator remains a `v0.x` project. Review release notes and upgrade guidance
before changing generator minor versions.

#!/usr/bin/env bash
set -euo pipefail
case_name="${1:?generation case is required}"
root="$(cd "$(dirname "$0")/.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
go build -o "$work/make-app" "$root"

new_project() {
  "$work/make-app" new "Matrix App" --module example.com/matrix-app --output "$work/app" "$@"
}

verify_project() {
  (cd "$work/app/apps/api" && go test ./... && go run ./cmd/openapi >"$work/openapi.json")
  test -s "$work/openapi.json"
}

case "$case_name" in
  default)
    new_project
    test -f "$work/app/apps/api/go.mod"
    verify_project
    ;;
  without-example)
    new_project --without-example
    test ! -e "$work/app/apps/api/internal/domain/example"
    verify_project
    ;;
  all-field-types)
    new_project --without-example
    "$work/make-app" domain add measurement --dir "$work/app" --fields 'title:string,active:bool,count:int,ratio:float,observed_at:time'
    verify_project
    ;;
  irregular-plural)
    new_project --without-example
    "$work/make-app" domain add person --dir "$work/app" --plural people
    grep -q '"plural": "people"' "$work/app/.make-app.json"
    verify_project
    ;;
  multiple-domains)
    new_project --without-example
    "$work/make-app" domain add foo_1 --dir "$work/app"
    "$work/make-app" domain add foo1 --dir "$work/app"
    verify_project
    ;;
  example-removal)
    new_project
    "$work/make-app" example remove --dir "$work/app"
    test ! -e "$work/app/apps/api/internal/domain/example"
    verify_project
    ;;
  schema-compatibility)
    new_project --without-example
    python3 - "$work/app/.make-app.json" <<'PY'
import json,sys
path=sys.argv[1]
data=json.load(open(path))
data['schemaVersion']=3
open(path,'w').write(json.dumps(data))
PY
    if "$work/make-app" domain add refused --dir "$work/app" 2>"$work/error"; then exit 1; fi
    grep -q 'upgrading-v3-to-v4.md' "$work/error"
    verify_project
    ;;
  identifier-boundaries)
    name="A$(printf 'b%.0s' {1..79})"
    "$work/make-app" new "$name" --module example.com/boundary --output "$work/app" --without-example
    domain="d$(printf 'x%.0s' {1..39})"
    "$work/make-app" domain add "$domain" --dir "$work/app"
    verify_project
    ;;
  existing-repository-adoption)
    mkdir "$work/app"
    git -C "$work/app" init -b main
    mkdir -p "$work/app/specs/habits"
    printf '# Existing\n' >"$work/app/specs/habits/habits.spec.md"
    "$work/make-app" init "Adopted App" --module example.com/adopted --dir "$work/app" --without-example
    test -f "$work/app/specs/habits/habits.spec.md"
    verify_project
    ;;
  *) echo "unknown generation case: $case_name" >&2; exit 2 ;;
esac

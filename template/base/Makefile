.PHONY: bootstrap generate check test race dependency-age security verify pre-commit pre-push acceptance infra-up dev api web mobile mobile-export mobile-validate mobile-prebuild mobile-build-android mobile-build-ios mobile-dev-android mobile-dev-ios mobile-release-android mobile-release-ios logs db-shell migrate audit-retention reset seed compose-up compose-down

bootstrap:
	@command -v go >/dev/null && command -v node >/dev/null && command -v pnpm >/dev/null && command -v python3 >/dev/null || (echo "missing prerequisite; run make-app doctor" >&2; exit 1)
	test -f .env || cp .env.example .env
	cd apps/api && go mod tidy
	pnpm install --frozen-lockfile
	$(MAKE) generate
	$(MAKE) check
	test -f pnpm-lock.yaml
	@echo "Ready: web http://localhost:5173 | API http://localhost:8080 | docs http://localhost:8080/docs"
	@echo "Next: make dev (web/API) or make mobile (Expo)"

generate:
	cd apps/api && go run ./cmd/openapi > ../../packages/api-client/openapi.json
	pnpm --dir packages/api-client generate

check:
	./scripts/plan-release_test.sh
	./scripts/check-structure.sh
	cd apps/api && test -z "$$(gofmt -l .)"
	cd apps/api && go test ./...
	pnpm check
	pnpm test
	node scripts/validate-mobile-config.mjs
	cd apps/mobile && ../../tools/eas-cli/node_modules/.bin/eas --version

test:
	cd apps/api && go test ./...
	pnpm test

race:
	cd apps/api && go test -race ./...

dependency-age:
	python3 scripts/test-dependency-age.py
	python3 scripts/check-dependency-age.py

security:
	mkdir -p .bin
	cd tools && go build -o ../.bin/govulncheck golang.org/x/vuln/cmd/govulncheck
	cd apps/api && ../../.bin/govulncheck ./...
	pnpm audit --audit-level low
	python3 scripts/check-ruby-vulnerabilities.py

verify: check race dependency-age security
	$(MAKE) generate
	git diff --exit-code -- packages/api-client
	pnpm build

pre-commit: check
	$(MAKE) generate
	git diff --exit-code -- packages/api-client
	@if git diff --cached --name-only | grep -Eq '(^|/)(go\.mod|go\.sum|package\.json|pnpm-lock\.yaml|Gemfile|Gemfile\.lock|Dockerfile[^/]*|.*\.ya?ml|dependency-age-allowlist\.json)$$'; then $(MAKE) dependency-age security; else echo "dependency graph unchanged; age/security graph scan deferred to pre-push and CI"; fi

pre-push: verify acceptance

acceptance:
	pnpm exec playwright install chromium-headless-shell
	./scripts/live-acceptance.sh .

compose-up:
	MAKE_APP_UID=$$(id -u) MAKE_APP_GID=$$(id -g) docker compose up --build

compose-down:
	docker compose down

infra-up:
	docker compose up -d postgres dex spicedb-runtime-proxy
	docker compose run --rm app-migrate

dev:
	./scripts/dev.sh

api:
	./scripts/run-api.sh

web:
	pnpm --dir apps/web dev --host 127.0.0.1

mobile:
	set -a; . ./.env; set +a; pnpm --dir apps/mobile start

mobile-export:
	set -a; . ./.env; set +a; pnpm --dir apps/mobile mobile:export

mobile-validate:
	pnpm --dir apps/mobile mobile:doctor
	pnpm --dir apps/mobile mobile:compat
	node scripts/validate-mobile-config.mjs

mobile-prebuild: mobile-validate
	pnpm --dir apps/mobile mobile:prebuild

mobile-build-android: mobile-validate
	pnpm --dir apps/mobile mobile:build:android

mobile-build-ios: mobile-validate
	pnpm --dir apps/mobile mobile:build:ios

mobile-dev-android:
	set -a; . ./.env; set +a; pnpm --dir apps/mobile mobile:dev:android

mobile-dev-ios:
	set -a; . ./.env; set +a; pnpm --dir apps/mobile mobile:dev:ios

mobile-release-android:
	set -a; . ./.env; set +a; APP_ENV=production node scripts/validate-mobile-release-env.mjs
	set -a; . ./.env; set +a; APP_ENV=production pnpm --dir apps/mobile mobile:release:android

mobile-release-ios:
	set -a; . ./.env; set +a; APP_ENV=production node scripts/validate-mobile-release-env.mjs
	set -a; . ./.env; set +a; APP_ENV=production pnpm --dir apps/mobile mobile:release:ios

logs:
	docker compose logs -f --tail=200

db-shell:
	docker compose exec postgres psql -U app_migrator -d app

migrate:
	docker compose run --rm app-migrate

audit-retention:
	docker compose --profile operations run --rm audit-retention

reset:
	@test "$(RESET)" = "1" || (echo "refusing destructive reset; run make reset RESET=1" >&2; exit 1)
	docker compose down --volumes --remove-orphans

seed:
	./scripts/seed.sh

.PHONY: bootstrap generate check test dependency-age security verify acceptance compose-up compose-down

bootstrap:
	cd apps/api && go mod tidy
	pnpm install --frozen-lockfile=false
	$(MAKE) generate
	$(MAKE) check
	test -f pnpm-lock.yaml

generate:
	cd apps/api && go run ./cmd/openapi > ../../packages/api-client/openapi.json
	pnpm --dir packages/api-client generate

check:
	./scripts/check-structure.sh
	cd apps/api && test -z "$$(gofmt -l .)"
	cd apps/api && go test ./...
	pnpm check
	pnpm test

test:
	cd apps/api && go test ./...
	pnpm test

dependency-age:
	python3 scripts/test-dependency-age.py
	python3 scripts/check-dependency-age.py

security:
	mkdir -p .bin
	cd tools && go build -o ../.bin/govulncheck golang.org/x/vuln/cmd/govulncheck
	cd apps/api && ../../.bin/govulncheck ./...
	pnpm audit --audit-level low

verify: check dependency-age security
	$(MAKE) generate
	git diff --exit-code -- packages/api-client
	pnpm build

acceptance:
	pnpm exec playwright install chromium-headless-shell
	./scripts/live-acceptance.sh .

compose-up:
	MAKE_APP_UID=$$(id -u) MAKE_APP_GID=$$(id -g) docker compose up --build

compose-down:
	docker compose down

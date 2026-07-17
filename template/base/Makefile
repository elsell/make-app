.PHONY: bootstrap generate check test dependency-age security verify compose-up compose-down

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

test:
	cd apps/api && go test ./...
	pnpm test

dependency-age:
	python3 scripts/test-dependency-age.py
	python3 scripts/check-dependency-age.py

security:
	cd apps/api && go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
	pnpm audit --audit-level low

verify: check dependency-age security
	$(MAKE) generate
	git diff --exit-code -- packages/api-client
	pnpm build

compose-up:
	docker compose up --build

compose-down:
	docker compose down

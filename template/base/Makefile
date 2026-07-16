.PHONY: bootstrap generate check test compose-up compose-down

bootstrap:
	cd apps/api && go mod tidy
	pnpm install --frozen-lockfile=false
	$(MAKE) generate
	$(MAKE) check

generate:
	cd apps/api && go run ./cmd/openapi > ../../packages/api-client/openapi.json
	pnpm --dir packages/api-client generate

check:
	cd apps/api && test -z "$$(gofmt -l .)"
	cd apps/api && go test ./...
	pnpm check

test:
	cd apps/api && go test ./...
	pnpm test

compose-up:
	docker compose up --build

compose-down:
	docker compose down

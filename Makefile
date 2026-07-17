.PHONY: test dependency-age security verify acceptance

test:
	go test -race ./...
	python3 scripts/test-dependency-age.py

dependency-age:
	python3 scripts/check-dependency-age.py

security:
	mkdir -p .bin
	cd tools && go build -o ../.bin/govulncheck golang.org/x/vuln/cmd/govulncheck
	./.bin/govulncheck ./...

verify: test dependency-age security

acceptance:
	scripts/acceptance.sh

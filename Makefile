.PHONY: test dependency-age security verify acceptance

test:
	go test -race ./...
	python3 scripts/test-dependency-age.py

dependency-age:
	python3 scripts/check-dependency-age.py

security:
	go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...

verify: test dependency-age security

acceptance:
	scripts/acceptance.sh

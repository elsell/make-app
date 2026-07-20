.PHONY: test dependency-age security verify acceptance

test:
	go test -race ./...
	scripts/plan-release_test.sh
	scripts/release-docs_test.sh
	scripts/publish-release_test.sh
	scripts/check-release-docs.sh
	python3 scripts/test-dependency-age.py

dependency-age:
	python3 scripts/check-dependency-age.py

security:
	mkdir -p .bin
	cd tools && go build -o ../.bin/govulncheck golang.org/x/vuln/cmd/govulncheck
	./.bin/govulncheck ./...
	python3 scripts/check-ruby-vulnerabilities.py

verify: test dependency-age security

acceptance:
	scripts/acceptance.sh

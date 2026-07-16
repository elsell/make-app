.PHONY: test acceptance

test:
	go test -race ./...

acceptance:
	scripts/acceptance.sh

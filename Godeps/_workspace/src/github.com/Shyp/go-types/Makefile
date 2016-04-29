.PHONY: install build test

install:
	go install ./...

build:
	go build ./...

test:
	go test ./...

release:
	bump_version minor types.go

.PHONY: install test

install:
	go install ./...

build:
	go get ./...
	go build ./...

test: install
	go test -v -race ./...

release:
	bump_version minor metrics.go

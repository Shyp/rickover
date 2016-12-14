.PHONY: install test race-test

SHELL = /bin/bash

ifdef DATABASE_URL
DATABASE_URL = $(DATABASE_URL)
TEST_DATABASE_URL = $(DATABASE_URL)
else
DATABASE_URL = 'postgres://rickover@localhost:5432/rickover?sslmode=disable&timezone=UTC'
TEST_DATABASE_URL = 'postgres://rickover@localhost:5432/rickover_test?sslmode=disable&timezone=UTC'
endif

BENCHSTAT := $(shell command -v benchstat)

test-install: 
	-createuser rickover --superuser --createrole --createdb --inherit
	-createdb rickover --owner=rickover
	-createdb rickover_test --owner=rickover

lint:
	go vet ./...

build:
	go build ./...

install:
	go install ./...

docs:
	go get golang.org/x/tools/cmd/godoc
	(sleep 1; open http://localhost:6060/pkg/github.com/Shyp/rickover) &
	godoc -http=:6060

testonly: 
	@DATABASE_URL=$(TEST_DATABASE_URL) go test -p 1 ./... -timeout 2s

race-testonly:
	@DATABASE_URL=$(TEST_DATABASE_URL) go test -p 1 -race -v ./... -timeout 2s

truncate-test:
	@DATABASE_URL=$(TEST_DATABASE_URL) rickover-truncate-tables

race-test: install race-testonly truncate-test

test: install testonly truncate-test

serve:
	@DATABASE_URL=$(DATABASE_URL) go run commands/server/main.go

dequeue:
	@DATABASE_URL=$(DATABASE_URL) go run commands/dequeuer/main.go

release: race-test
	go get github.com/Shyp/bump_version
	bump_version minor config/config.go
	git push origin master
	git push origin master --tags

migrate:
	goose --env=development up
	goose --env=test up

bench:
ifndef BENCHSTAT
	go get -u rsc.io/benchstat
endif
	tmp=$$(mktemp); go list ./... | grep -v vendor | xargs go test -benchtime=2s -bench=. -run='^$$' > "$$tmp" 2>&1 && benchstat "$$tmp"

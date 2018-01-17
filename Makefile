.PHONY: install test race-test

SHELL = /bin/bash -x -o pipefail

ifdef DATABASE_URL
	DATABASE_URL := $(DATABASE_URL)
	TEST_DATABASE_URL := $(DATABASE_URL)
else
	DATABASE_URL := 'postgres://rickover@localhost:5432/rickover?sslmode=disable&timezone=UTC'
	TEST_DATABASE_URL := 'postgres://rickover@localhost:5432/rickover_test?sslmode=disable&timezone=UTC'
endif

BENCHSTAT := $(GOPATH)/bin/benchstat
BUMP_VERSION := $(GOPATH)/bin/bump_version
GODOCDOC := $(GOPATH)/bin/godocdoc
GOOSE := $(GOPATH)/bin/goose

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

$(GODOCDOC):
	go get -u github.com/kevinburke/godocdoc

docs: | $(GODOCDOC)
	$(GODOCDOC)

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

$(BUMP_VERSION):
	go get -u github.com/Shyp/bump_version

release: race-test | $(BUMP_VERSION)
	$(BUMP_VERSION) minor config/config.go
	git push origin master
	git push origin master --tags

GOOSE:
	go get -u github.com/kevinburke/goose/cmd/goose

migrate: | $(GOOSE)
	$(GOOSE) --env=development up
	$(GOOSE) --env=test up

$(BENCHSTAT):
	go get -u golang.org/x/perf/cmd/benchstat

bench: | $(BENCHSTAT)
	tmp=$$(mktemp); go list ./... | grep -v vendor | xargs go test -benchtime=2s -bench=. -run='^$$' > "$$tmp" 2>&1 && $(BENCHSTAT) "$$tmp"

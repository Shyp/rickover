.PHONY: test race-test

SHELL = /bin/bash -o pipefail

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
TRUNCATE_TABLES := $(GOPATH)/bin/rickover-truncate-tables

# Just run it every time, we could get fancy with find() tricks, but eh.
$(TRUNCATE_TABLES):
	go install ./test/rickover-truncate-tables

test-install:
	-createuser rickover --superuser --createrole --createdb --inherit
	-createdb rickover --owner=rickover
	-createdb rickover_test --owner=rickover

lint:
	go list ./... | grep -v vendor | xargs go vet

build:
	go build ./...

$(GODOCDOC):
	go get -u github.com/kevinburke/godocdoc

docs: | $(GODOCDOC)
	$(GODOCDOC)

testonly:
	@DATABASE_URL=$(TEST_DATABASE_URL) go list ./... | grep -v vendor | xargs go test -p=1 -timeout 10s

race-testonly:
	DATABASE_URL=$(TEST_DATABASE_URL) go list ./... | grep -v vendor | xargs go test -p=1 -race -timeout 10s

truncate-test: $(TRUNCATE_TABLES)
	@DATABASE_URL=$(TEST_DATABASE_URL) $(TRUNCATE_TABLES)

race-test: race-testonly truncate-test

test: testonly truncate-test

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
	tmp=$$(mktemp); go list ./... | grep -v vendor | xargs go test -p=1 -benchtime=2s -bench=. -run='^$$' > "$$tmp" 2>&1 && $(BENCHSTAT) "$$tmp"

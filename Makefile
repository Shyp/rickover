.PHONY: install test race-test

ifdef DATABASE_URL
DATABASE_URL = $(DATABASE_URL)
TEST_DATABASE_URL = $(DATABASE_URL)
else
DATABASE_URL = 'postgres://rickover@localhost:5432/rickover?sslmode=disable&timezone=UTC'
TEST_DATABASE_URL = 'postgres://rickover@localhost:5432/rickover_test?sslmode=disable&timezone=UTC'
endif

test-install:
	-createuser rickover --superuser --createrole --createdb --inherit
	-createdb rickover --owner=rickover
	-createdb rickover_test --owner=rickover

build:
	go build ./...

docs:
	go get golang.org/x/tools/cmd/godoc
	(sleep 1; open http://localhost:6060/pkg/github.com/Shyp/rickover) &
	godoc -http=:6060

test: 
	@DATABASE_URL=$(TEST_DATABASE_URL) go test ./... -timeout 2s

race-test:
	@DATABASE_URL=$(TEST_DATABASE_URL) go test -race -v ./... -timeout 2s

serve:
	@DATABASE_URL=$(DATABASE_URL) go run commands/server/main.go

dequeue:
	@DATABASE_URL=$(DATABASE_URL) go run commands/dequeuer/main.go

release:
	go get github.com/Shyp/bump_version
	bump_version minor config/config.go
	git push origin master
	git push origin master --tags

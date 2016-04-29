.PHONY: install test race-test

ifndef DATABASE_URL
DATABASE_URL = 'postgres://rickover@localhost:5432/rickover_test?sslmode=disable&timezone=UTC'
endif

test-install:
	-createuser rickover --superuser --createrole --createdb --inherit
	-createdb rickover --owner=shyp_jobs
	-createdb rickover_test --owner=shyp_jobs

build:
	go build ./...

docs:
	go get golang.org/x/tools/cmd/godoc
	(sleep 1; open http://localhost:6060/pkg/github.com/Shyp/rickover) &
	godoc -http=:6060

test: 
	@DATABASE_URL=$(DATABASE_URL) go test ./config/... ./dequeuer/... ./downstream/... ./models/... ./rest/... ./server/... ./services/... ./setup/... -timeout 2s
	@DATABASE_URL=$(DATABASE_URL) DEPLOYMENT_NAME=test go test -p 1 ./test/... -timeout 2s

race-test:
	@DATABASE_URL=$(DATABASE_URL) go test -race -v ./config/... ./dequeuer/... ./downstream/... ./models/... ./rest/... ./server/... ./services/... ./setup/... -timeout 2s
	@DATABASE_URL=$(DATABASE_URL) DEPLOYMENT_NAME=test go test -race -p 1 -v ./test/... -timeout 2s

serve:
	go run commands/server/main.go

dequeue:
	go run commands/dequeuer/main.go

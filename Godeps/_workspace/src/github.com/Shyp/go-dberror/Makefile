.PHONY: install test

install:
	go install ./...

test:
	go test -race ./... -timeout 2s

test-install: 
	-createdb dberror
	go get -u bitbucket.org/liamstask/goose/cmd/goose
	go get -u github.com/letsencrypt/boulder/test
	goose up

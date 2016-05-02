.PHONY: install build test

install:
	go install ./...

build:
	go build ./...

test:
	go test ./...

release:
	go get github.com/Shyp/bump_version
	bump_version minor types.go

docs:
	go get golang.org/x/tools/cmd/godoc
	(sleep 1; open http://localhost:6060/pkg/github.com/Shyp/go-types) &
	godoc -http=:6060

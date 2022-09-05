BINARY=athena

all: build test

build:
		go build -v -o bin/${BINARY} athena.go
release:
		goreleaser --skip-publish --rm-dist
test:
		go test -v ./...
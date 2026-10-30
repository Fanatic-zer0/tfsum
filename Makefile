BINARY     := tfsum
MODULE     := github.com/Fanatic-zer0/tfsum
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS    := -X '$(MODULE)/internal/cli.Version=$(VERSION)'

.PHONY: all build install test test-verbose cover vet fmt fmt-check lint clean

all: build

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/tfsum

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/tfsum

test:
	go test ./...

test-verbose:
	go test -v ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needs to be run on:"; gofmt -l .; exit 1)

lint: fmt-check vet

clean:
	rm -rf bin coverage.out

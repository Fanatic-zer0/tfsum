BINARY     := tfsum
MODULE     := github.com/dheeryad/tfsum
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS    := -X '$(MODULE)/internal/cli.Version=$(VERSION)'

.PHONY: all build install test test-verbose cover golden vet fmt fmt-check lint clean

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

# Regenerate golden fixtures in testdata/golden after an intentional
# rendering change. Review the diff before committing.
golden:
	UPDATE_GOLDEN=1 go test ./internal/render/...

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needs to be run on:"; gofmt -l .; exit 1)

lint: fmt-check vet

clean:
	rm -rf bin coverage.out

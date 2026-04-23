REPO := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

GO = go

.PHONY: all build dav-mcp clean

all: deps build

build: dav-mcp

deps:
	$(GO) mod tidy
	$(GO) mod download

get:
	$(GO) get -v ./...

dav-mcp:
	$(GO) build -o bin/dav-mcp -v ./cmd/dav-mcp

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

test:
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html

check: fmt vet test

clean:
	$(GO) clean -v
	rm -rf $(REPO)/pkg $(REPO)/bin

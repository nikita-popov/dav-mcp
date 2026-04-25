REPO := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

GO = go

# git describe: v0.1.0 on a tagged commit, v0.1.0-3-gabcdef on later commits.
# Append -dev so local builds are always distinguishable from CI releases.
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/$/-dev/')
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all build dav-mcp clean

all: deps build

build: dav-mcp

deps:
	$(GO) mod tidy
	$(GO) mod download

get:
	$(GO) get -v ./...

dav-mcp:
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o bin/dav-mcp ./cmd/dav-mcp

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

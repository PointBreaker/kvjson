GO ?= go
BINARY ?= bin/kv
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= unknown
MODULE := github.com/PointBreaker/kvjson
LDFLAGS := -X $(MODULE)/internal/buildinfo.Version=$(VERSION) -X $(MODULE)/internal/buildinfo.Commit=$(COMMIT) -X $(MODULE)/internal/buildinfo.Date=$(DATE)

.PHONY: build test check fmt

build:
	mkdir -p $(dir $(BINARY))
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/kv

test:
	$(GO) test ./...

check:
	@files="$$(gofmt -l .)"; if [ -n "$$files" ]; then echo "Files need gofmt:"; echo "$$files"; exit 1; fi
	$(GO) vet ./...
	$(GO) test -race ./...
	mkdir -p $(dir $(BINARY))
	$(GO) build -o $(BINARY) ./cmd/kv

fmt:
	gofmt -w .

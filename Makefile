BIN      := typist
PKG      := ./cmd/typist
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w -X main.version=$(VERSION)
DEV_DB   := .dev/typist.db

.PHONY: dev dev-stats dev-db build install test vet clean

## dev: run from source against the throwaway dev database
dev:
	go run $(PKG) --db $(DEV_DB)

## dev-stats: open the stats screen against the dev database
dev-stats:
	go run $(PKG) --db $(DEV_DB) stats

## dev-db: poke at the dev database with sqlite3
dev-db:
	sqlite3 $(DEV_DB)

## build: compile a release binary into ./bin
build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BIN) $(PKG)

## install: install typist into $(go env GOPATH)/bin for everyday use (real database)
install:
	go install -trimpath -ldflags "$(LDFLAGS)" $(PKG)

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf bin .dev

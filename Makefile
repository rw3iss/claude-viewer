.PHONY: build install dev clean test fmt lint cross release

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
  -X github.com/rw3iss/claude-viewer/internal/version.Version=$(VERSION) \
  -X github.com/rw3iss/claude-viewer/internal/version.Commit=$(COMMIT) \
  -X github.com/rw3iss/claude-viewer/internal/version.Date=$(DATE)

BIN := claude-viewer
DEST ?= $(HOME)/.local/bin

build:
	go build -ldflags '$(LDFLAGS)' -o bin/$(BIN) ./cmd/claude-viewer

install: build
	mkdir -p $(DEST)
	install -m 0755 bin/$(BIN) $(DEST)/$(BIN)
	@echo "installed: $(DEST)/$(BIN)"

dev:
	go run ./cmd/claude-viewer

test:
	go test ./...

fmt:
	go fmt ./...

lint:
	go vet ./...

clean:
	rm -rf bin dist

# Cross-compile for common platforms.
cross:
	mkdir -p dist
	GOOS=linux   GOARCH=amd64 go build -ldflags '$(LDFLAGS)' -o dist/$(BIN)-linux-amd64    ./cmd/claude-viewer
	GOOS=linux   GOARCH=arm64 go build -ldflags '$(LDFLAGS)' -o dist/$(BIN)-linux-arm64    ./cmd/claude-viewer
	GOOS=darwin  GOARCH=amd64 go build -ldflags '$(LDFLAGS)' -o dist/$(BIN)-darwin-amd64   ./cmd/claude-viewer
	GOOS=darwin  GOARCH=arm64 go build -ldflags '$(LDFLAGS)' -o dist/$(BIN)-darwin-arm64   ./cmd/claude-viewer
	GOOS=windows GOARCH=amd64 go build -ldflags '$(LDFLAGS)' -o dist/$(BIN)-windows-amd64.exe ./cmd/claude-viewer
	@echo "see dist/"

release:
	goreleaser release --clean

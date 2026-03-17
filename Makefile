.PHONY: help lint lint-fix lint-check test build clean run-api run-websocket

GOLANGCI_LINT_VERSION ?= v2.11.3
GOLANGCI_LINT_BIN := $(CURDIR)/bin/golangci-lint

# Default target
help:
	@echo "Available targets:"
	@echo "  lint         - Run golangci-lint on the entire codebase"
	@echo "  lint-fix     - Run golangci-lint with auto-fix enabled"
	@echo "  lint-check   - Run golangci-lint and exit with error if issues found (for CI)"
	@echo "  test         - Run all tests"
	@echo "  build        - Build the API and WebSocket servers"
	@echo "  build-api    - Build the API server"
	@echo "  build-ws     - Build the WebSocket server"
	@echo "  clean        - Remove build artifacts"
	@echo "  run-api      - Run the API server"
	@echo "  run-websocket - Run the WebSocket server"

# Linting targets
lint-install:
	@GOBIN="$(CURDIR)/bin" go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

lint: lint-install
	@"$(GOLANGCI_LINT_BIN)" run ./...

lint-fix: lint-install
	@"$(GOLANGCI_LINT_BIN)" run --fix ./...

lint-check: lint-install
	@"$(GOLANGCI_LINT_BIN)" run ./... || (echo "Linting failed. Fix issues or run 'make lint-fix' to auto-fix some issues." && exit 1)

# Testing
test:
	@go test ./...

test-verbose:
	@go test -v ./...

test-coverage:
	@go test -cover ./...

# Build targets
build: build-api build-ws

build-api:
	@echo "Building API server..."
	@go build -o bin/api cmd/api/main.go

build-ws:
	@echo "Building WebSocket server..."
	@go build -o bin/websocket cmd/websocket/main.go

build-linux:
	@echo "Building for Linux..."
	@GOOS=linux GOARCH=amd64 go build -o bin/api-linux cmd/api/main.go
	@GOOS=linux GOARCH=amd64 go build -o bin/websocket-linux cmd/websocket/main.go

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@go clean

# Run targets
run-api:
	@go run local/cmd/server.go

run-websocket:
	@go run cmd/websocket/main.go

# Format code
fmt:
	@go fmt ./...

# Vet code
vet:
	@go vet ./...

# All checks (lint, vet, test)
check: lint vet test
	@echo "All checks passed!"


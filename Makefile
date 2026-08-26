.PHONY: help test test-verbose test-race test-cover codecov bench fmt lint clean check-fmt vet

# Build variables
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO := go
GOFLAGS ?=
GOBIN := $(shell go env GOPATH)/bin

## help: Display this help message
help:
	@echo "Mattermost Logr - Makefile Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@grep -E '^## ' Makefile | sed 's/##/  /'

## test: Run all tests
test:
	$(GO) test $(GOFLAGS) ./...

## test-verbose: Run all tests with verbose output
test-verbose:
	$(GO) test $(GOFLAGS) -v ./...

## test-race: Run tests with race detector
test-race:
	$(GO) test $(GOFLAGS) -race ./...

## test-cover: Run tests with coverage report
test-cover:
	$(GO) test $(GOFLAGS) -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## codecov: Run tests with coverage and display summary
codecov:
	$(GO) test $(GOFLAGS) -coverprofile=coverage.out -covermode=atomic ./...
	@echo ""
	@echo "Coverage Summary:"
	@$(GO) tool cover -func=coverage.out
	@echo ""
	@echo "Generating HTML report..."
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "HTML coverage report: coverage.html"

## bench: Run benchmarks
bench:
	$(GO) test $(GOFLAGS) -bench=. ./test

## fmt: Format all Go files
fmt:
	$(GO) fmt ./...

## check-fmt: Check if Go files are formatted
check-fmt:
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "The following files are not formatted:"; \
		gofmt -l .; \
		exit 1; \
	fi

## vet: Run go vet on all packages
vet:
	$(GO) vet ./...

## lint: Run golangci-lint
lint:
	@test -f $(GOBIN)/golangci-lint || { echo "golangci-lint not installed. Installing..."; $(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0; }
	$(GOBIN)/golangci-lint run ./...

## check: Run format check, vet, lint, and tests
check: check-fmt vet lint test

## clean: Remove generated files
clean:
	$(GO) clean
	rm -f coverage.out coverage.html

## version: Display version information
version:
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Build Hash: $(BUILD_HASH)"

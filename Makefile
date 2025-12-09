.PHONY: all build test test-unit lint lint-fix clean coverage check help gameid

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test

# Default target
all: lint test build

# Build the project
build:
	@echo "Building packages..."
	$(GOBUILD) -v ./...

# Build gameid binary
gameid:
	@echo "Building gameid..."
	$(GOBUILD) -o cmd/gameid/gameid ./cmd/gameid

# Run all tests with race detection
test: test-unit
	@echo "All tests completed!"

# Run unit tests with race detection
test-unit:
	@echo "Running unit tests..."
	$(GOTEST) -v -race -timeout=10m ./...

# Run tests with coverage report
coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated at coverage.html"

# Run linters
lint:
	@echo "Running linters..."
	$(GOCMD) mod tidy
	golangci-lint run ./...

# Run linters with auto-fix
lint-fix:
	@echo "Running linters with auto-fix..."
	$(GOCMD) mod tidy
	golangci-lint run --fix ./...

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCMD) clean
	rm -f coverage.txt coverage.html
	rm -rf bin/ dist/ build/
	rm -f cmd/gameid/gameid

# Quick check before committing
check: lint test
	@echo "All checks passed!"

# Show help
help:
	@echo "go-gameid Makefile"
	@echo "=================="
	@echo ""
	@echo "Available targets:"
	@echo "  all              - Lint, test, and build (default)"
	@echo "  build            - Build all packages"
	@echo "  gameid           - Build gameid binary to cmd/gameid/"
	@echo "  test             - Run all tests with race detection"
	@echo "  test-unit        - Run unit tests with race detection"
	@echo "  bench            - Run benchmarks"
	@echo "  coverage         - Run tests and generate HTML coverage report"
	@echo "  lint             - Run linters (golangci-lint)"
	@echo "  lint-fix         - Run linters with auto-fix"
	@echo "  clean            - Remove build artifacts and coverage files"
	@echo "  check            - Run lint and test (pre-commit check)"
	@echo "  help             - Show this help message"

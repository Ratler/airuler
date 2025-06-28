# airuler - AI Rules Template Engine
# Makefile for building, testing, and development

# Project settings
BINARY_NAME=airuler
MAIN_PACKAGE=.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT_HASH=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go settings
GO=go
GOFMT=gofmt
GOLINT=golangci-lint
GOTEST=$(GO) test
GOBUILD=$(GO) build
GOCLEAN=$(GO) clean
GOMOD=$(GO) mod

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.CommitHash=$(COMMIT_HASH)"

# Directories
DIST_DIR=dist
COVERAGE_DIR=coverage

# Default target
.PHONY: all
all: clean lint test build

# Build targets
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PACKAGE)

.PHONY: build-all
build-all: clean
	@echo "Building for multiple platforms..."
	@mkdir -p $(DIST_DIR)
	@echo "Building for Linux amd64..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	@echo "Building for Linux arm64..."
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PACKAGE)
	@echo "Building for macOS amd64..."
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	@echo "Building for macOS arm64..."
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)
	@echo "Building for Windows amd64..."
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)
	@echo "Built binaries:"
	@ls -la $(DIST_DIR)/

# Development targets
.PHONY: dev
dev: build
	@echo "Development build complete. Run ./$(BINARY_NAME) --help"

.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME) to $(GOPATH)/bin..."
	@cp $(BINARY_NAME) $(GOPATH)/bin/

# Testing targets
.PHONY: test
test:
	@echo "Running tests..."
	AIRULER_USE_MOCK_GIT=1 $(GOTEST) -v ./...

.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	AIRULER_USE_MOCK_GIT=1 $(GOTEST) -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report generated: $(COVERAGE_DIR)/coverage.html"

.PHONY: test-coverage-func
test-coverage-func:
	@echo "Running tests with function coverage..."
	@mkdir -p $(COVERAGE_DIR)
	AIRULER_USE_MOCK_GIT=1 $(GOTEST) -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	$(GO) tool cover -func=$(COVERAGE_DIR)/coverage.out

.PHONY: benchmark
benchmark:
	@echo "Running benchmarks..."
	AIRULER_USE_MOCK_GIT=1 $(GOTEST) -bench=. -benchmem ./...

# Code quality targets
.PHONY: lint
lint:
	@echo "Running linter..."
	@if command -v $(GOLINT) > /dev/null; then \
		$(GOLINT) run; \
	else \
		echo "golangci-lint not found. Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin v1.55.2"; \
		exit 1; \
	fi

.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

.PHONY: fmt-check
fmt-check:
	@echo "Checking code formatting..."
	@if [ -n "$$($(GOFMT) -l .)" ]; then \
		echo "The following files are not formatted:"; \
		$(GOFMT) -l .; \
		echo "Run 'make fmt' to fix formatting"; \
		exit 1; \
	fi

.PHONY: vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...

.PHONY: check
check: fmt-check vet lint

# Dependency management
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download

.PHONY: deps-update
deps-update:
	@echo "Updating dependencies..."
	$(GOMOD) tidy
	$(GOMOD) download

.PHONY: deps-vendor
deps-vendor:
	@echo "Vendoring dependencies..."
	$(GOMOD) vendor

# Cleanup targets
.PHONY: clean
clean:
	@echo "Cleaning up..."
	$(GOCLEAN)
	@rm -f $(BINARY_NAME)
	@rm -rf $(DIST_DIR)
	@rm -rf $(COVERAGE_DIR)
	@rm -rf vendor/

.PHONY: clean-all
clean-all: clean
	@echo "Deep cleaning..."
	$(GOMOD) clean -cache

# Development helpers
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

.PHONY: run-help
run-help: build
	@echo "Showing help..."
	./$(BINARY_NAME) --help

.PHONY: demo
demo: build
	@echo "Running demo..."
	@rm -rf demo-project
	@mkdir demo-project
	@cd demo-project && ../$(BINARY_NAME) init
	@cd demo-project && ../$(BINARY_NAME) compile
	@echo "Demo project created in demo-project/"

# CI/CD targets
.PHONY: ci
ci: deps check test-coverage build

.PHONY: release
release: clean check test build-all
	@echo "Release build complete. Binaries in $(DIST_DIR)/"

# Documentation
.PHONY: docs
docs: build
	@echo "Generating command documentation..."
	@mkdir -p docs/commands
	./$(BINARY_NAME) --help > docs/commands/help.txt
	./$(BINARY_NAME) init --help > docs/commands/init.txt
	./$(BINARY_NAME) compile --help > docs/commands/compile.txt
	./$(BINARY_NAME) install --help > docs/commands/install.txt
	./$(BINARY_NAME) fetch --help > docs/commands/fetch.txt
	./$(BINARY_NAME) update --help > docs/commands/update.txt
	./$(BINARY_NAME) vendors --help > docs/commands/vendors.txt
	./$(BINARY_NAME) config --help > docs/commands/config.txt
	./$(BINARY_NAME) watch --help > docs/commands/watch.txt
	@echo "Command documentation generated in docs/commands/"

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build           - Build the binary"
	@echo "  build-all       - Build for all platforms"
	@echo "  dev             - Development build"
	@echo "  install         - Install binary to GOPATH/bin"
	@echo ""
	@echo "  test            - Run tests"
	@echo "  test-coverage   - Run tests with HTML coverage report"
	@echo "  test-coverage-func - Run tests with function coverage"
	@echo "  benchmark       - Run benchmarks"
	@echo ""
	@echo "  lint            - Run linter"
	@echo "  fmt             - Format code"
	@echo "  fmt-check       - Check code formatting"
	@echo "  vet             - Run go vet"
	@echo "  check           - Run all checks (fmt, vet, lint)"
	@echo ""
	@echo "  deps            - Download dependencies"
	@echo "  deps-update     - Update dependencies"
	@echo "  deps-vendor     - Vendor dependencies"
	@echo ""
	@echo "  clean           - Clean build artifacts"
	@echo "  clean-all       - Deep clean including cache"
	@echo ""
	@echo "  run             - Build and run"
	@echo "  run-help        - Build and show help"
	@echo "  demo            - Create demo project"
	@echo ""
	@echo "  ci              - CI pipeline (deps, check, test, build)"
	@echo "  release         - Release build for all platforms"
	@echo "  docs            - Generate command documentation"
	@echo ""
	@echo "  help            - Show this help"
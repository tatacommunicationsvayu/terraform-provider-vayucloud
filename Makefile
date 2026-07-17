# Makefile for terraform-provider-vayucloud
#
# This Makefile provides convenient commands for building, testing,
# and developing the VayuCloud Terraform provider.

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt

# Binary name
BINARY_NAME=terraform-provider-vayucloud

# Version (can be overridden via environment variable)
VERSION?=1.0.0

# OS and Architecture detection
OS=$(shell go env GOOS)
ARCH=$(shell go env GOARCH)

# Installation directories
PLUGIN_DIR=~/.terraform.d/plugins
REGISTRY_PATH=registry.terraform.io/tatacommunicationsvayu/vayucloud/$(VERSION)/$(OS)_$(ARCH)

# Ldflags for version injection
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# Default target: build for current platform
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .
	@echo "Build complete: $(BINARY_NAME)"

# Build for all platforms
build-all: build-linux build-darwin build-windows

build-linux:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)_linux_amd64 .
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)_linux_arm64 .

build-darwin:
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)_darwin_amd64 .
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)_darwin_arm64 .

build-windows:
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)_windows_amd64.exe .

# Install the provider to the local plugin directory
install: build
	@echo "Installing provider to $(PLUGIN_DIR)/$(REGISTRY_PATH)..."
	@mkdir -p $(PLUGIN_DIR)/$(REGISTRY_PATH)
	@cp $(BINARY_NAME) $(PLUGIN_DIR)/$(REGISTRY_PATH)/
	@echo "Provider installed successfully"
	@echo ""
	@echo "You can now use the provider in your Terraform configurations."
	@echo "Make sure to run 'terraform init' in your Terraform directory."

# Uninstall the provider
uninstall:
	@echo "Removing provider from $(PLUGIN_DIR)/registry.terraform.io/tatacommunicationsvayu/vayucloud..."
	@rm -rf $(PLUGIN_DIR)/registry.terraform.io/tatacommunicationsvayu/vayucloud
	@echo "Provider uninstalled"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run acceptance tests (requires TF_ACC=1)
testacc:
	@echo "Running acceptance tests..."
	TF_ACC=1 $(GOTEST) -v ./... -timeout 120m

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .
	@echo "Code formatted"

# Check code formatting
fmt-check:
	@echo "Checking code formatting..."
	@test -z "$$($(GOFMT) -l .)" || (echo "Code is not formatted. Run 'make fmt'" && exit 1)
	@echo "Code formatting is correct"

# Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Dependencies downloaded"

# Generate documentation
docs:
	@echo "Generating documentation..."
	@if command -v tfplugindocs >/dev/null 2>&1; then \
		tfplugindocs generate; \
	else \
		echo "tfplugindocs not installed. Install it with: go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest"; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY_NAME)
	@rm -f $(BINARY_NAME)_*
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

# Development helper: watch for changes and rebuild
dev:
	@echo "Starting development mode..."
	@echo "Run 'make build' after making changes"
	@echo "Tip: Use 'dev_overrides' in ~/.terraformrc for instant testing"

# Create dev_overrides configuration
dev-setup:
	@echo "Setting up development environment..."
	@echo ""
	@echo "Add the following to your ~/.terraformrc (or %APPDATA%/terraform.rc on Windows):"
	@echo ""
	@echo 'provider_installation {'
	@echo '  dev_overrides {'
	@echo '    "tatacommunicationsvayu/vayucloud" = "$(shell pwd)"'
	@echo '  }'
	@echo '  direct {}'
	@echo '}'
	@echo ""

# Show help
help:
	@echo "VayuCloud Terraform Provider - Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build        Build the provider binary"
	@echo "  make install      Install provider to local Terraform plugin directory"
	@echo "  make uninstall    Remove provider from local plugin directory"
	@echo "  make test         Run unit tests"
	@echo "  make testacc      Run acceptance tests (requires TF_ACC=1)"
	@echo "  make fmt          Format Go code"
	@echo "  make lint         Run linter"
	@echo "  make docs         Generate documentation"
	@echo "  make deps         Download and tidy dependencies"
	@echo "  make clean        Remove build artifacts"
	@echo "  make dev-setup    Show dev_overrides configuration"
	@echo "  make help         Show this help message"
	@echo ""
	@echo "Environment variables:"
	@echo "  VERSION           Provider version (default: 1.0.0)"


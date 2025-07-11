# Kubernetes OIDC Authentication Plugin Makefile

.PHONY: build clean test run-mock-server help

# Build all authentication plugins
build: build-auth-plugin build-mock-server build-test-auth-plugin

# Build individual components
build-auth-plugin:
	@echo "Building full-featured auth plugin..."
	go build -o bin/auth-plugin cmd/auth-plugin/main.go


build-mock-server:
	@echo "Building mock OIDC server..."
	go build -o bin/mock-server cmd/mock-server/main.go

build-test-auth-plugin:
	@echo "Building test auth plugin..."
	go build -o bin/test-auth-plugin cmd/test-auth-plugin/main.go

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Run mock server
run-mock-server: build-mock-server
	@echo "Starting mock OIDC server..."
	./bin/mock-server

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy

# Create bin directory
bin:
	mkdir -p bin

# Help
help:
	@echo "Available targets:"
	@echo "  build                 - Build all components"
	@echo "  build-auth-plugin     - Build full-featured auth plugin"
	@echo "  build-mock-server     - Build mock OIDC server"
	@echo "  build-test-auth-plugin - Build test auth plugin"
	@echo "  clean                 - Clean build artifacts"
	@echo "  test                  - Run tests"
	@echo "  run-mock-server       - Start mock OIDC server"
	@echo "  deps                  - Install dependencies"
	@echo "  help                  - Show this help message"

# Default target
all: build
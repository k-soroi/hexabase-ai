# Hexabase AI Makefile
# Common development tasks

.PHONY: help setup clean dev-api dev-ui test lint build deploy-dev deploy-staging deploy-prod

# Default target
help:
	@echo "Hexabase AI Development Commands"
	@echo "=================================="
	@echo "Setup & Cleanup:"
	@echo "  make setup          - Set up complete development environment"
	@echo "  make clean          - Clean up development environment"
	@echo ""
	@echo "Development:"
	@echo "  make dev-api        - Run API server in development mode"
	@echo "  make dev-ui         - Run UI development server"
	@echo "  make dev            - Run both API and UI (requires tmux)"
	@echo ""
	@echo "Testing:"
	@echo "  make test           - Run all tests"
	@echo "  make test-api       - Run API tests"
	@echo "  make test-ui        - Run UI tests"
	@echo "  make test-e2e       - Run end-to-end tests"
	@echo "  make coverage       - Generate test coverage report"
	@echo ""
	@echo "Code Quality:"
	@echo "  make lint           - Run linters"
	@echo "  make fmt            - Format code"
	@echo ""
	@echo "Building:"
	@echo "  make build          - Build all components"
	@echo "  make docker         - Build Docker images"
	@echo ""
	@echo "Deployment:"
	@echo "  make deploy-dev     - Deploy to local development"
	@echo "  make deploy-staging - Deploy to staging"
	@echo "  make deploy-prod    - Deploy to production"
	@echo ""
	@echo "Debugging:"
	@echo "  make debug          - Start unified debug environment"
	@echo "  make debug-api      - Debug API with Delve"
	@echo "  make debug-ui       - Debug UI with Chrome DevTools"
	@echo "  make debug-e2e      - Debug E2E tests interactively"
	@echo "  make debug-e2e-dev  - Debug E2E tests in developer mode"
	@echo "  make debug-basic    - Run debug basic functions test"
	@echo "  make debug-logs     - Stream all service logs with filtering"
	@echo "  make debug-status   - Show debug environment status"
	@echo "  make debug-stop     - Stop debug environment"
	@echo ""
	@echo "Utilities:"
	@echo "  make status         - Show environment status"
	@echo "  make logs-api       - Tail API logs"
	@echo "  make db-shell       - Access database shell"
	@echo "  make migrate-create - Interactively create a new migration file"

# Setup development environment
setup:
	@echo "Setting up development environment..."
	@./scripts/dev-setup.sh

# Clean up development environment
clean:
	@echo "Cleaning up development environment..."
	@./scripts/dev-cleanup.sh

# Run API server
dev-api:
	@echo "Running database migrations..."
	@docker compose -f docker-compose.yml --profile tools run --rm migrate
	@echo "Starting API server..."
	@cd api && if [ -f .env ]; then export $$(cat .env | grep -v '^#' | xargs); fi && go run cmd/api/main.go

# Run UI development server
dev-ui:
	@echo "Starting UI development server..."
	@cd ui && npm install && npm run dev

# Run both API and UI using tmux
dev:
	@if command -v tmux >/dev/null 2>&1; then \
		tmux new-session -d -s hexabase-dev 'make dev-api' \; \
			split-window -h 'make dev-ui' \; \
			attach-session -t hexabase-dev; \
	else \
		echo "tmux not found. Please install tmux or run 'make dev-api' and 'make dev-ui' in separate terminals."; \
	fi

# Run all tests
test: test-api test-ui

# Run API tests
test-api:
	@echo "Running API tests..."
	@cd api && go test ./... -v

# Run UI tests
test-ui:
	@echo "Running UI tests..."
	@cd ui && npm test

# Run end-to-end tests
test-e2e:
	@echo "Running end-to-end tests..."
	@cd ui && npm run test:e2e

# Generate coverage report
coverage:
	@echo "Running API tests with coverage..."
	@cd api && go test ./... -coverprofile=coverage.out
	@cd api && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at api/coverage.html"

# Run linters
lint:
	@echo "Running Go linter..."
	@cd api && golangci-lint run
	@echo "Running UI linters..."
	@cd ui && npm run lint

# Run API linter
lint-api:
	@echo "Running Go linter for API..."
	@cd api && go install tool && go tool golangci-lint version && go tool golangci-lint run -c ./.golangci.yaml --new-from-merge-base origin/develop

# Format code
fmt:
	@echo "Formatting Go code..."
	@cd api && go fmt ./...
	@echo "Formatting UI code..."
	@cd ui && npm run format

# Build all components
build:
	@echo "Building API binary..."
	@cd api && go build -o bin/hexabase-api cmd/api/main.go
	@echo "Building UI for production..."
	@cd ui && npm run build

# Build Docker images
docker:
	@echo "Building Docker images..."
	@docker build -t hexabase/hexabase-ai-api:latest -f api/Dockerfile api/
	@docker build -t hexabase/hexabase-ai-ui:latest -f ui/Dockerfile ui/

# Deploy to development
deploy-dev:
	@echo "Deploying to development..."
	@helm upgrade --install hexabase-ai ./deployments/helm/hexabase-ai \
		--namespace hexabase-dev \
		--create-namespace \
		--values deployments/helm/values-local.yaml \
		--wait

# Deploy to staging
deploy-staging:
	@echo "Deploying to staging..."
	@helm upgrade --install hexabase-ai ./deployments/helm/hexabase-ai \
		--namespace hexabase-staging \
		--create-namespace \
		--values deployments/helm/values-staging.yaml \
		--wait

# Deploy to production
deploy-prod:
	@echo "⚠️  WARNING: You are about to deploy to PRODUCTION!"
	@echo -n "Are you sure? Type 'yes' to continue: "
	@read confirm && [ "$$confirm" = "yes" ] || (echo "Deployment cancelled"; exit 1)
	@echo "Deploying to production..."
	@helm upgrade --install hexabase-ai ./deployments/helm/hexabase-ai \
		--namespace hexabase-system \
		--create-namespace \
		--values deployments/helm/values-production.yaml \
		--wait

# Show environment status
status:
	@echo "Environment Status:"
	@echo "=================="
	@echo "Docker Compose:"
	@docker-compose ps
	@echo ""
	@echo "Kind Cluster:"
	@kind get clusters
	@echo ""
	@echo "Kubernetes Pods:"
	@kubectl get pods -A | grep hexabase || echo "No hexabase pods found"

# Tail logs
logs-api:
	@docker-compose logs -f postgres redis nats

# Database shell
db-shell:
	@docker-compose exec postgres psql -U hexabase -d hexabase_kaas

# Create a new migration file
migrate-create:
	@read -p "Enter migration name in snake_case (e.g., add_user_api_keys): " name; \
	if [ -z "$$name" ]; then \
		echo "Error: Migration name cannot be empty."; \
		exit 1; \
	fi; \
	cd api && go tool migrate create -ext sql -dir internal/shared/db/migrations -format "20060102150405000" $$name

# Quick start (alias for setup)
start: setup

# Quick stop (alias for clean)
stop: clean

# Debug targets
debug:
	@echo "Starting unified debug environment..."
	@./scripts/unified-debug.sh start

debug-api:
	@echo "Starting API in debug mode..."
	@docker compose -f docker-compose.yml -f docker-compose.debug.yml up -d postgres redis nats
	@echo "Running database migrations..."
	@docker compose -f docker-compose.yml -f docker-compose.debug.yml --profile tools run --rm migrate
	@docker compose -f docker-compose.yml -f docker-compose.debug.yml up api

debug-ui:
	@echo "Starting UI in debug mode..."
	@./scripts/unified-debug.sh start
	@echo "UI debugger available at chrome://inspect"

debug-e2e:
	@echo "Running E2E tests in debug mode..."
	@./scripts/e2e-debug-enhanced.sh

debug-e2e-dev:
	@echo "Running E2E tests in developer mode..."
	@./scripts/e2e-debug-enhanced.sh --developer

debug-basic:
	@echo "Running debug basic functions test..."
	@./scripts/e2e-debug-enhanced.sh --developer --test debug-basic-functions.spec.ts

debug-logs:
	@echo "Streaming debug logs..."
	@./scripts/unified-debug.sh logs

debug-status:
	@echo "Showing debug environment status..."
	@./scripts/unified-debug.sh status

debug-stop:
	@echo "Stopping debug environment..."
	@./scripts/unified-debug.sh stop

debug-restart:
	@echo "Restarting debug environment..."
	@./scripts/unified-debug.sh restart
# Development Environment Setup

This guide will help you set up a complete development environment for Hexabase KaaS.

## Quick Setup (Automated)

We provide an automated setup script that handles most of the configuration for you:

```bash
# Clone the repository
git clone https://github.com/hexabase/hexabase-ai.git
cd hexabase-ai

# Run the setup script
./scripts/dev-setup.sh
```

The script will:
1. Check for required dependencies
2. Create a Kind cluster with ingress support
3. Install necessary Kubernetes components
4. Start PostgreSQL, Redis, and NATS using Docker Compose
5. Generate JWT keys
6. Create .env files with proper configuration
7. Update /etc/hosts for local access

After running the script, you can start developing:

```bash
# Terminal 1: Start the API
cd api
go run cmd/api/main.go

# Terminal 2: Start the UI
cd ui
npm install
npm run dev
```

Access the application at:
- API: http://api.localhost
- UI: http://app.localhost

## Manual Setup (Alternative)

If you prefer to set up the environment manually or need to understand what the script does, follow these steps:

### Prerequisites

### Required Software

1. **Go** (1.24 or later)
   ```bash
   # macOS
   brew install go
   
   # Linux
   wget https://go.dev/dl/go1.24.3.linux-amd64.tar.gz
   sudo tar -C /usr/local -xzf go1.24.3.linux-amd64.tar.gz
   export PATH=$PATH:/usr/local/go/bin
   ```

2. **Node.js** (18.x or later) and npm
   ```bash
   # macOS
   brew install node
   
   # Linux (using NodeSource)
   curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
   sudo apt-get install -y nodejs
   ```

3. **Docker** and Docker Compose
   ```bash
   # macOS
   brew install --cask docker
   
   # Linux
   curl -fsSL https://get.docker.com | sh
   sudo usermod -aG docker $USER
   ```

4. **Kubernetes Tools**
   ```bash
   # kubectl
   brew install kubectl
   
   # k3s (lightweight Kubernetes)
   curl -sfL https://get.k3s.io | sh -
   
   # OR kind (Kubernetes in Docker)
   brew install kind
   ```

5. **PostgreSQL Client**
   ```bash
   # macOS
   brew install postgresql
   
   # Linux
   sudo apt-get install postgresql-client
   ```

6. **Redis Client**
   ```bash
   # macOS
   brew install redis
   
   # Linux
   sudo apt-get install redis-tools
   ```

### Development Tools

1. **Wire** (Dependency Injection)
   ```bash
   go install github.com/google/wire/cmd/wire@latest
   ```

2. **golang-migrate** (Database Migrations)
   ```bash
   brew install golang-migrate
   # OR
   go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
   ```

3. **golangci-lint** (Linting)
   ```bash
   cd api
   go install tool
   ```

4. **Playwright** (E2E Testing)
   ```bash
   cd ui
   npm install
   npx playwright install
   ```

## Setting Up the Development Environment

### 1. Clone the Repository

```bash
git clone https://github.com/hexabase/hexabase-ai.git
cd hexabase-ai
```

### 2. Start Infrastructure Services

Create a `docker-compose.override.yml` for local development:

```yaml
version: '3.8'

services:
  postgres:
    ports:
      - "5432:5432"
    environment:
      POSTGRES_PASSWORD: localdev

  redis:
    ports:
      - "6379:6379"

  nats:
    image: nats:latest
    ports:
      - "4222:4222"
      - "8222:8222"
    command: "-js"
```

Start the services:

```bash
docker-compose up -d postgres redis nats
```

### 3. Set Up the Database

```bash
# Create database
psql -h localhost -U postgres -c "CREATE DATABASE hexabase_kaas;"

# Run migrations
cd api
migrate -path ./migrations -database "postgresql://postgres:localdev@localhost:5432/hexabase_kaas?sslmode=disable" up
```

### 4. Configure Environment Variables

Create `.env` files for both API and UI:

**api/.env**
```env
# Database
DATABASE_URL=postgres://postgres:localdev@localhost:5432/hexabase_kaas?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379

# NATS
NATS_URL=nats://localhost:4222

# JWT Keys (generate with: openssl genrsa -out private.pem 2048)
JWT_PRIVATE_KEY_PATH=./keys/private.pem
JWT_PUBLIC_KEY_PATH=./keys/public.pem

# OAuth Providers (replace with your values)
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret

# Stripe (for billing)
STRIPE_API_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...

# Kubernetes
KUBECONFIG=$HOME/.kube/config
```

**ui/.env.local**
```env
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_WS_URL=ws://localhost:8080
```

### 5. Generate JWT Keys

```bash
cd api
mkdir -p keys
openssl genrsa -out keys/private.pem 2048
openssl rsa -in keys/private.pem -pubout -out keys/public.pem
```

### 6. Set Up Local Kubernetes Cluster

Using k3s:
```bash
# Install k3s (if not already installed)
curl -sfL https://get.k3s.io | sh -

# Copy kubeconfig
mkdir -p ~/.kube
sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
sudo chown $USER:$USER ~/.kube/config
```

OR using kind:
```bash
# Create cluster
kind create cluster --name hexabase-dev

# Install vCluster operator
kubectl create namespace vcluster
kubectl apply -f https://github.com/loft-sh/vcluster/releases/latest/download/vcluster-k8s.yaml
```

### 7. Build and Run the API

```bash
cd api

# Download dependencies
go mod download

# Generate wire dependencies
wire ./internal/infrastructure/wire

# Run tests
go test ./...

# Run the API server
go run cmd/api/main.go
```

### 8. Build and Run the UI

```bash
cd ui

# Install dependencies
npm install

# Run development server
npm run dev
```

## Development Workflow

### Running Tests

**API Tests:**
```bash
cd api
go test ./...                    # Run all tests
go test ./... -cover            # With coverage
go test -v ./internal/auth/...  # Specific package
```

**UI Tests:**
```bash
cd ui
npm test                        # Unit tests
npm run test:e2e               # Playwright E2E tests
```

### Code Quality Checks

**API:**
```bash
# Linting (recommended - checks only new changes)
make lint-api

# Or run directly for new changes only
cd api && go tool golangci-lint run -c ./.golangci.yaml --new-from-merge-base origin/develop

# Run linting for all files (not just new changes)
cd api && go tool golangci-lint run -c ./.golangci.yaml

# Run all linters (both Go and UI)
make lint

# Format code
go fmt ./...

# Vet code
go vet ./...
```

**UI:**
```bash
# Linting
npm run lint

# Type checking
npm run type-check

# Format code
npm run format
```

### Database Migrations

```bash
# Create a new migration
migrate create -ext sql -dir api/migrations -seq add_new_table

# Apply migrations
migrate -path api/migrations -database $DATABASE_URL up

# Rollback
migrate -path api/migrations -database $DATABASE_URL down 1
```

## IDE Setup

### VS Code

Install recommended extensions:
- Go (official)
- ESLint
- Prettier
- GitLens
- Docker
- Kubernetes

Create `.vscode/settings.json`:
```json
{
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "workspace",
  "editor.formatOnSave": true,
  "[go]": {
    "editor.defaultFormatter": "golang.go"
  },
  "[typescript]": {
    "editor.defaultFormatter": "esbenp.prettier-vscode"
  }
}
```

### JetBrains GoLand/WebStorm

- Enable Go modules support
- Configure JavaScript version to ES2022
- Set up file watchers for formatting

## Troubleshooting

### Common Issues

1. **Port already in use**
   ```bash
   # Find process using port
   lsof -i :8080
   # Kill process
   kill -9 <PID>
   ```

2. **Database connection errors**
   - Ensure PostgreSQL is running: `docker-compose ps`
   - Check credentials in `.env`
   - Verify database exists: `psql -h localhost -U postgres -l`

3. **Kubernetes connection issues**
   - Check cluster is running: `kubectl cluster-info`
   - Verify kubeconfig: `kubectl config current-context`

4. **Go module errors**
   ```bash
   go clean -modcache
   go mod download
   ```

## Next Steps

- Read the [API Development Guide](./api-development-guide.md)
- Explore the [Frontend Development Guide](./frontend-development-guide.md)
- Review [Code Style Guide](./code-style-guide.md)
- Understand the [Git Workflow](./git-workflow.md)
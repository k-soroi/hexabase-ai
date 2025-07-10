# Development Environment Setup

This guide will help you set up a local development environment for Hexabase KaaS.

## Quick Start

The easiest way to set up your development environment is to use the automated setup script:

```bash
./scripts/dev-setup.sh
```

This script will:
- Install missing dependencies (kind, helm, kubectl) automatically
- Create a local Kubernetes cluster using Kind
- Set up infrastructure services (PostgreSQL, Redis, NATS)
- Configure the development environment
- Generate necessary configuration files

## Prerequisites

### Required Dependencies

The following tools must be installed on your system:

#### Core Requirements
- **Docker** - Container runtime
  - macOS: [Docker Desktop](https://docs.docker.com/desktop/install/mac-install/)
  - Linux: [Docker Engine](https://docs.docker.com/engine/install/)
  
- **Go** (1.21+) - Backend development
  - Installation: https://golang.org/doc/install
  
- **Node.js** (18+) - Frontend development
  - Installation: https://nodejs.org/

#### Auto-Installable Tools

The setup script will automatically install these if missing:

- **kubectl** - Kubernetes CLI
- **kind** - Local Kubernetes clusters
- **helm** - Kubernetes package manager

### Optional Tools

- **vCluster** - Virtual Kubernetes clusters (installed by setup script)
- **k9s** - Terminal UI for Kubernetes
- **kubectx/kubens** - Kubernetes context/namespace switcher

## Manual Installation

If you prefer to install dependencies manually:

### macOS (using Homebrew)

```bash
# Install Homebrew if not already installed
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install dependencies
brew install docker
brew install go
brew install node@18
brew install kubectl
brew install kind
brew install helm
```

### macOS (without Homebrew)

```bash
# Install Docker Desktop
# Download from: https://docs.docker.com/desktop/install/mac-install/

# Install Go
# Download from: https://golang.org/dl/

# Install Node.js
# Download from: https://nodejs.org/

# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/darwin/$(uname -m)/kubectl"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/

# Install kind
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-darwin-$(uname -m)
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

# Install helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

### Linux (Ubuntu/Debian)

```bash
# Install Docker (includes Docker Compose)
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER  # Add user to docker group
newgrp docker  # Activate group change

# Install Go
wget https://go.dev/dl/go1.21.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc

# Install Node.js
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt-get install -y nodejs

# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/

# Install kind
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

# Install helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

## Development Workflow

After running the setup script, follow these steps:

### 1. Start the API Server

```bash
cd api
go run cmd/api/main.go
```

The API will be available at:
- Local: http://localhost:8080
- Via Ingress: http://api.localhost

### 2. Start the UI Development Server

```bash
cd ui
npm install  # First time only
npm run dev
```

The UI will be available at:
- Local: http://localhost:3000
- Via Ingress: http://app.localhost

### 3. Access Infrastructure Services

The services use non-standard ports to avoid conflicts. Check your `.env` file for the actual ports, or use these defaults:

- **PostgreSQL**: `localhost:5433`
  - User: `postgres`
  - Password: `postgres`
  - Database: `hexabase`

- **Redis**: `localhost:6380`

- **NATS**: `localhost:4223`
  - Monitoring: http://localhost:8223

### 4. Kubernetes Access

```bash
# Use the development cluster
kubectl config use-context kind-hexabase-dev

# List all pods
kubectl get pods -A

# Access cluster with k9s (if installed)
k9s --context kind-hexabase-dev
```

## Troubleshooting

### Permission Denied Errors

If you get permission errors when installing tools:
```bash
# Add sudo to move commands
sudo mv ./kind /usr/local/bin/kind

# Or use a user-writable location
mkdir -p ~/.local/bin
mv ./kind ~/.local/bin/
export PATH=$PATH:~/.local/bin
```

### Docker Not Running

Ensure Docker is running:
```bash
# macOS
open -a Docker

# Linux
sudo systemctl start docker
```

### Kind Cluster Issues

```bash
# Delete and recreate cluster
kind delete cluster --name hexabase-dev
./scripts/dev-setup.sh
```

### Port Conflicts

The setup script automatically finds available ports. If you still have conflicts:

1. **Check what's using the ports:**
```bash
# Find process using port
sudo lsof -i :5433  # PostgreSQL
sudo lsof -i :6380  # Redis
sudo lsof -i :4223  # NATS
sudo lsof -i :8080  # API
```

2. **Modify the `.env` file to use different ports:**
```bash
# Edit .env file
vim .env

# Change ports as needed:
POSTGRES_HOST_PORT=5434
REDIS_HOST_PORT=6381
NATS_HOST_PORT=4224
API_HOST_PORT=8081
```

3. **Restart services:**
```bash
# Stop services
docker compose down  # or docker-compose down

# Start with new ports
docker compose up -d
```

## Clean Up

To completely remove the development environment:

```bash
# Stop services
docker compose down  # or docker-compose down

# Delete Kind cluster
kind delete cluster --name hexabase-dev

# Remove generated files
rm -rf api/.env api/keys
rm -f ui/.env.local
rm -f docker-compose.override.yml
```

## Next Steps

- Read the [Architecture Overview](../architecture/README.md)
- Check out the [API Development Guide](./api-development.md)
- Learn about [UI Development](./ui-development.md)
- Review [Testing Guidelines](./testing.md)
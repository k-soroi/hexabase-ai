#!/bin/bash

# Run API in Docker with live logs

echo "Starting Hexabase API in Docker..."
echo "API will be available at http://localhost:8085"
echo "Press Ctrl+C to stop"
echo ""

# Stop any existing containers
docker-compose down

# Start dependencies first
echo "Starting dependencies (PostgreSQL, Redis, NATS)..."
docker-compose up -d postgres redis nats

# Wait for dependencies to be ready
echo "Waiting for dependencies to be ready..."
sleep 5

# Run migrations
echo "Running database migrations..."
docker-compose run --rm migrate || echo "Migration issues detected, continuing..."

# Load OAuth credentials if available
if [ -f .env.oauth ]; then
    echo "Loading OAuth credentials from .env.oauth..."
    export $(cat .env.oauth | grep -v '^#' | xargs)
else
    echo "WARNING: .env.oauth not found. Google OAuth will not work."
    echo "Copy .env.oauth.example to .env.oauth and add your credentials."
fi

# Build and run API with logs
echo "Building and starting API server..."
export API_HOST_PORT=8085
docker-compose -f docker-compose.yml -f docker-compose.dev.yml up --build api
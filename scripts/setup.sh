#!/bin/bash
# =============================================================================
# OmniDev AI Platform — Development Environment Setup
# =============================================================================

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# Check prerequisites
check_command() {
    if ! command -v "$1" &> /dev/null; then
        error "$1 is not installed. Please install it first."
    fi
    info "$1 found: $(command -v "$1")"
}

info "Checking prerequisites..."
check_command go
check_command node
check_command pnpm
check_command docker
check_command make

# Check Go version
GO_VERSION=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+')
if [[ $(echo "$GO_VERSION < 1.22" | bc) -eq 1 ]]; then
    error "Go 1.22+ required, found $GO_VERSION"
fi
info "Go version: $GO_VERSION"

# Check Node version
NODE_VERSION=$(node -v | grep -oP 'v\K[0-9]+')
if [[ $NODE_VERSION -lt 20 ]]; then
    error "Node.js 20+ required, found v$NODE_VERSION"
fi
info "Node.js version: v$NODE_VERSION"

# Copy environment file
if [ ! -f .env ]; then
    cp .env.example .env
    info "Created .env from .env.example"
else
    warn ".env already exists, skipping"
fi

# Install Go dependencies
info "Installing Go dependencies..."
go mod download all
go work sync

# Install frontend dependencies
info "Installing frontend dependencies..."
pnpm install

# Start infrastructure
info "Starting infrastructure services..."
docker compose -f deploy/docker/docker-compose.infra.yml up -d

# Wait for services
info "Waiting for services to be ready..."
sleep 10

# Check services
info "Checking service health..."
docker compose -f deploy/docker/docker-compose.infra.yml ps

# Run migrations
info "Running database migrations..."
# TODO: implement migrate command

info ""
info "=========================================="
info " Setup complete!"
info "=========================================="
info ""
info " Next steps:"
info "   1. Edit .env with your API keys"
info "   2. Run 'make run-gateway' to start the API"
info "   3. Run 'make web-dev' to start the frontend"
info ""
info " Services:"
info "   Gateway:    http://localhost:9090"
info "   Web:        http://localhost:3000"
info "   MinIO:      http://localhost:9001"
info "   Temporal:   http://localhost:8088"
info ""

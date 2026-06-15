#!/bin/bash
# =============================================================================
# FreeRADIUS Manager - First-Time Setup Script
# =============================================================================
set -euo pipefail

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

info() { echo -e "${BLUE}[INFO]${NC} $*"; }
success() { echo -e "${GREEN}[OK]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }

echo ""
echo -e "${BLUE}=================================================${NC}"
echo -e "${BLUE}   FreeRADIUS Manager - Setup Script v1.0.0     ${NC}"
echo -e "${BLUE}=================================================${NC}"
echo ""

# --- Check prerequisites ---
info "Checking prerequisites..."

command -v docker >/dev/null 2>&1 || error "Docker is not installed. Install it from https://docs.docker.com/get-docker/"
command -v docker-compose >/dev/null 2>&1 || {
    # Try 'docker compose' (v2)
    docker compose version >/dev/null 2>&1 || error "Docker Compose is not installed."
    COMPOSE_CMD="docker compose"
}
COMPOSE_CMD="${COMPOSE_CMD:-docker-compose}"

success "Docker and Docker Compose found"

# Check if Docker daemon is running
docker info >/dev/null 2>&1 || error "Docker daemon is not running. Start Docker and try again."
success "Docker daemon is running"

# --- Generate secure passwords ---
info "Generating secure passwords..."

generate_password() {
    openssl rand -base64 32 | tr -d '/+=' | head -c 40
}

generate_secret() {
    openssl rand -hex 32
}

# Check if .env already exists
if [ -f ".env" ]; then
    warn ".env file already exists. Skipping password generation."
    warn "If you want to regenerate passwords, delete .env and re-run setup."
else
    DB_PASS=$(generate_password)
    JWT_SECRET=$(generate_secret)
    RADIUS_SECRET=$(generate_password | head -c 32)

    cat > .env << EOF
# ============================================================
# FreeRADIUS Manager - Auto-generated Configuration
# Generated on: $(date)
# ============================================================

# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=radius_user
DB_PASSWORD=${DB_PASS}
DB_NAME=radius
DB_SSL_MODE=disable

# JWT Authentication
JWT_SECRET=${JWT_SECRET}
JWT_ACCESS_EXPIRY=1h
JWT_REFRESH_EXPIRY=168h

# RADIUS Server
RADIUS_HOST=freeradius
RADIUS_AUTH_PORT=1812
RADIUS_ACCT_PORT=1813
RADIUS_SECRET=${RADIUS_SECRET}
RADIUS_TIMEOUT=5s
RADIUS_RETRIES=3

# Web Interface
WEB_PORT=8080
VITE_API_URL=/api/v1

# Security
ADMIN_IP_WHITELIST=0.0.0.0/0
SESSION_TIMEOUT=3600
BCRYPT_COST=12
RATE_LIMIT=100
BRUTE_FORCE_ATTEMPTS=5
BRUTE_FORCE_LOCKOUT=15

# Application
APP_ENV=production

# Email (configure if needed)
SMTP_HOST=
SMTP_PORT=587
SMTP_USER=
SMTP_PASS=
SMTP_FROM=noreply@radius-manager.local

# Backup
BACKUP_SCHEDULE=0 2 * * *
BACKUP_RETENTION_DAYS=30
EOF

    success ".env file created with secure credentials"
fi

# --- Build and start services ---
echo ""
info "Building Docker images (this may take 3-10 minutes)..."
$COMPOSE_CMD build --parallel

echo ""
info "Starting services..."
$COMPOSE_CMD up -d

# --- Wait for services to be healthy ---
echo ""
info "Waiting for services to initialize..."

MAX_WAIT=120
ELAPSED=0

wait_for_health() {
    local service=$1
    local check_cmd=$2
    local max=${3:-60}
    local elapsed=0

    while ! eval "$check_cmd" 2>/dev/null; do
        if [ $elapsed -ge $max ]; then
            return 1
        fi
        sleep 3
        elapsed=$((elapsed + 3))
        echo -n "."
    done
    echo ""
    return 0
}

echo -n "  Waiting for PostgreSQL"
if wait_for_health "postgres" "$COMPOSE_CMD exec -T postgres pg_isready -U radius_user -d radius" 60; then
    success "PostgreSQL is ready"
else
    warn "PostgreSQL health check timed out. Check logs: $COMPOSE_CMD logs postgres"
fi

echo -n "  Waiting for backend API"
if wait_for_health "backend" "curl -sf http://localhost:8080/health" 60; then
    success "Backend API is ready"
else
    warn "Backend health check timed out. Check logs: $COMPOSE_CMD logs backend"
fi

echo -n "  Waiting for frontend"
if wait_for_health "frontend" "curl -sf http://localhost:8081/nginx-health" 30; then
    success "Frontend is ready"
else
    warn "Frontend health check timed out. Check logs: $COMPOSE_CMD logs frontend"
fi

# --- Send test RADIUS packet ---
echo ""
info "Sending test RADIUS packet..."
if $COMPOSE_CMD exec -T freeradius radtest testuser testpassword 127.0.0.1 0 testing123 2>&1 | grep -q "Access-Reject\|Access-Accept"; then
    success "FreeRADIUS server is responding"
else
    warn "FreeRADIUS test packet: server may still be starting. Check logs: $COMPOSE_CMD logs freeradius"
fi

# --- Display summary ---
echo ""
echo -e "${GREEN}=================================================${NC}"
echo -e "${GREEN}   Setup Complete!                               ${NC}"
echo -e "${GREEN}=================================================${NC}"
echo ""
echo -e "  ${BLUE}Web Interface:${NC}  http://localhost:8081"
echo -e "  ${BLUE}Backend API:${NC}    http://localhost:8080"
echo -e "  ${BLUE}RADIUS Auth:${NC}    localhost:1812/UDP"
echo -e "  ${BLUE}RADIUS Acct:${NC}    localhost:1813/UDP"
echo ""
echo -e "  ${YELLOW}Default Login:${NC}"
echo -e "  Username: superadmin"
echo -e "  Password: Admin@123456"
echo ""
echo -e "  ${RED}⚠  CHANGE THE DEFAULT PASSWORD IMMEDIATELY!${NC}"
echo ""
echo -e "  Useful commands:"
echo -e "  ${BLUE}View logs:${NC}    $COMPOSE_CMD logs -f"
echo -e "  ${BLUE}Stop:${NC}         $COMPOSE_CMD down"
echo -e "  ${BLUE}Restart:${NC}      $COMPOSE_CMD restart"
echo ""

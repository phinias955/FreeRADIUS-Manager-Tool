#!/bin/bash
# =============================================================================
# FreeRADIUS Manager - Production Update Script
# Pulls latest code, runs DB migrations, rebuilds containers.
# Usage: ./deploy-update.sh
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

COMPOSE_CMD="${COMPOSE_CMD:-docker-compose}"
command -v docker >/dev/null 2>&1 || error "Docker not installed"
$COMPOSE_CMD version >/dev/null 2>&1 || COMPOSE_CMD="docker compose"

cd "$(dirname "$0")"
info "Updating FreeRADIUS Manager in $(pwd)"

# Load DB vars from .env safely (do not source — cron values break bash)
if [ -f .env ]; then
  DB_USER="${DB_USER:-$(grep -E '^DB_USER=' .env | cut -d= -f2- | tr -d '\r' || true)}"
  DB_NAME="${DB_NAME:-$(grep -E '^DB_NAME=' .env | cut -d= -f2- | tr -d '\r' || true)}"
fi
DB_USER="${DB_USER:-radius_user}"
DB_NAME="${DB_NAME:-radius}"

# --- 1. Backup database ---
BACKUP_DIR="./backups"
mkdir -p "$BACKUP_DIR"
BACKUP_FILE="$BACKUP_DIR/radius_pre_update_$(date +%Y%m%d_%H%M%S).sql"
if docker ps --format '{{.Names}}' | grep -q '^radius_postgres$'; then
  info "Backing up database to $BACKUP_FILE ..."
  docker exec radius_postgres pg_dump -U "${DB_USER:-radius_user}" "${DB_NAME:-radius}" > "$BACKUP_FILE"
  success "Database backup saved"
else
  warn "Postgres container not running — skipping backup"
fi

# --- 2. Pull latest code (keep local .env) ---
info "Pulling latest code from origin/main ..."
git fetch origin main
git checkout main 2>/dev/null || true
BEFORE=$(git rev-parse HEAD)
git pull origin main
AFTER=$(git rev-parse HEAD)
if [ "$BEFORE" = "$AFTER" ]; then
  info "Already on latest commit: $(git log -1 --oneline)"
else
  success "Updated: $(git log -1 --oneline)"
fi

# --- 3. Run SQL migrations ---
if [ -d migrations ] && docker ps --format '{{.Names}}' | grep -q '^radius_postgres$'; then
  info "Applying database migrations ..."
  for f in $(ls migrations/*.sql 2>/dev/null | sort); do
    info "  → $f"
    docker exec -i radius_postgres psql -U "${DB_USER:-radius_user}" -d "${DB_NAME:-radius}" < "$f" || true
  done
  success "Migrations applied"
fi

# --- 4. Rebuild and restart (always rebuild — git pull alone is not enough) ---
info "Rebuilding containers (this may take a few minutes) ..."
$COMPOSE_CMD build --no-cache backend frontend freeradius

info "Restarting application containers ..."
docker rm -f radius_backend radius_frontend radius_freeradius 2>/dev/null || true
$COMPOSE_CMD up -d postgres freeradius backend frontend

# --- 5. Wait for health ---
info "Waiting for services to start ..."
sleep 12

if curl -sf http://localhost:8088/health >/dev/null 2>&1; then
  success "Backend health check passed (port 8088)"
else
  warn "Backend health check failed — check: docker logs radius_backend"
fi

if curl -sf http://localhost:8081/ >/dev/null 2>&1; then
  success "Frontend is responding (port 8081)"
else
  warn "Frontend not responding yet — check: docker logs radius_frontend"
fi

# Verify new backend code is running
if docker exec radius_backend strings /app/radius-manager 2>/dev/null | grep -q 'RevokeSession'; then
  success "Backend running updated session code"
else
  warn "Backend may still be on old code — rebuild may have failed"
fi

echo ""
success "Update complete!"
echo ""
echo "  Web UI:    http://$(hostname -I 2>/dev/null | awk '{print $1}'):8081"
echo "  API:       http://$(hostname -I 2>/dev/null | awk '{print $1}'):8088"
echo "  Version:   $(git log -1 --oneline)"
echo ""
warn "Hard-refresh your browser (Ctrl+Shift+R) after updating to load new JS."
echo ""

#!/bin/bash
set -e

echo "[FreeRADIUS] Waiting for PostgreSQL to be ready..."
until pg_isready -h "${DB_HOST:-postgres}" -p "${DB_PORT:-5432}" -U "${DB_USER:-radius_user}" 2>/dev/null; do
    echo "[FreeRADIUS] PostgreSQL not ready, retrying in 2s..."
    sleep 2
done
echo "[FreeRADIUS] PostgreSQL is ready."

# Safe replacement: escapes \ & | so any password/secret can be used
sed_escape() { printf '%s' "$1" | sed 's/\\/\\\\/g; s/&/\\&/g; s/|/\\|/g'; }

# ── 1. RADIUS shared secret → clients.conf ───────────────────────────────────
SECRET="${RADIUS_SECRET:-testing123}"
sed -i "s|RADIUS_SECRET_PLACEHOLDER|$(sed_escape "$SECRET")|g" \
    /etc/freeradius/clients.conf
echo "[FreeRADIUS] clients.conf secret set (length: ${#SECRET})."

# ── 2. DB credentials → mods-available/sql ───────────────────────────────────
SQL=/etc/freeradius/mods-available/sql
sed -i "s|DB_HOST_PLACEHOLDER|$(sed_escape "${DB_HOST:-postgres}")|g"    "$SQL"
sed -i "s|DB_PORT_PLACEHOLDER|$(sed_escape "${DB_PORT:-5432}")|g"        "$SQL"
sed -i "s|DB_NAME_PLACEHOLDER|$(sed_escape "${DB_NAME:-radius}")|g"      "$SQL"
sed -i "s|DB_USER_PLACEHOLDER|$(sed_escape "${DB_USER:-radius_user}")|g" "$SQL"
sed -i "s|DB_PASS_PLACEHOLDER|$(sed_escape "${DB_PASS:-}")|g"            "$SQL"
echo "[FreeRADIUS] SQL module credentials set (host: ${DB_HOST:-postgres})."

echo "[FreeRADIUS] Starting FreeRADIUS..."
exec freeradius -f -l stdout "$@"

#!/bin/bash
set -e

echo "[FreeRADIUS] Waiting for PostgreSQL to be ready..."
until pg_isready -h "${DB_HOST:-postgres}" -p "${DB_PORT:-5432}" -U "${DB_USER:-radius_user}" 2>/dev/null; do
    echo "[FreeRADIUS] PostgreSQL not yet ready, waiting..."
    sleep 2
done
echo "[FreeRADIUS] PostgreSQL is ready."

# Substitute environment variables in FreeRADIUS SQL module config
sed -i "s|\${DB_HOST}|${DB_HOST:-postgres}|g" /etc/freeradius/3.0/mods-available/sql
sed -i "s|\${DB_PORT}|${DB_PORT:-5432}|g" /etc/freeradius/3.0/mods-available/sql
sed -i "s|\${DB_NAME}|${DB_NAME:-radius}|g" /etc/freeradius/3.0/mods-available/sql
sed -i "s|\${DB_USER}|${DB_USER:-radius_user}|g" /etc/freeradius/3.0/mods-available/sql
sed -i "s|\${DB_PASS}|${DB_PASS:-}|g" /etc/freeradius/3.0/mods-available/sql

echo "[FreeRADIUS] Starting FreeRADIUS server..."
exec freeradius -f -l stdout "$@"

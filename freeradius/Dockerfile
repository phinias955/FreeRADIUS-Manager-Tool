# Use pinned FreeRADIUS 3.2.10
FROM freeradius/freeradius-server:3.2.10

USER root

# Install PostgreSQL client for the pg_isready healthcheck in init.sh
RUN apt-get update && apt-get install -y --no-install-recommends \
        postgresql-client \
    && rm -rf /var/lib/apt/lists/*

# ── Overwrite config files at the CORRECT path: /etc/freeradius/ ─────────────
COPY radiusd.conf                              /etc/freeradius/radiusd.conf
COPY clients.conf                              /etc/freeradius/clients.conf
COPY sites-available/default                   /etc/freeradius/sites-available/default
COPY sites-available/inner-tunnel              /etc/freeradius/sites-available/inner-tunnel
COPY mods-available/sql                        /etc/freeradius/mods-available/sql
COPY mods-config/sql/main/postgresql/queries.conf \
     /etc/freeradius/mods-config/sql/main/postgresql/queries.conf

COPY init.sh /init.sh

# Enable the SQL module (symlink in mods-enabled)
RUN ln -sf /etc/freeradius/mods-available/sql \
           /etc/freeradius/mods-enabled/sql && \
    chmod +x /init.sh

EXPOSE 1812/udp 1813/udp 18120/udp

ENTRYPOINT ["/init.sh"]

# Use pinned FreeRADIUS 3.2.10 (Debian-based, includes rlm_sql_postgresql)
FROM freeradius/freeradius-server:3.2.10

USER root

# Install PostgreSQL client for the healthcheck/startup script
RUN apt-get update && apt-get install -y --no-install-recommends \
        postgresql-client \
    && rm -rf /var/lib/apt/lists/*

# Copy custom configuration files
COPY radiusd.conf         /etc/freeradius/3.0/radiusd.conf
COPY clients.conf         /etc/freeradius/3.0/clients.conf
COPY sites-available/default      /etc/freeradius/3.0/sites-available/default
COPY sites-available/inner-tunnel /etc/freeradius/3.0/sites-available/inner-tunnel
COPY mods-available/sql           /etc/freeradius/3.0/mods-available/sql
COPY mods-config/sql/main/postgresql/queries.conf \
     /etc/freeradius/3.0/mods-config/sql/main/postgresql/queries.conf
COPY init.sh /init.sh

# Enable required modules and sites (symlinks)
RUN ln -sf /etc/freeradius/3.0/mods-available/sql \
           /etc/freeradius/3.0/mods-enabled/sql && \
    ln -sf /etc/freeradius/3.0/sites-available/default \
           /etc/freeradius/3.0/sites-enabled/default && \
    ln -sf /etc/freeradius/3.0/sites-available/inner-tunnel \
           /etc/freeradius/3.0/sites-enabled/inner-tunnel && \
    rm -f /etc/freeradius/3.0/sites-enabled/inner-tunnel.default 2>/dev/null || true && \
    chmod +x /init.sh

EXPOSE 1812/udp 1813/udp 18120/udp

ENTRYPOINT ["/init.sh"]

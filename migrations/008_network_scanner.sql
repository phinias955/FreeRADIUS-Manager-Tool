-- Network scanner tables (run on existing installations)
CREATE TABLE IF NOT EXISTS network_scans (
    id            SERIAL PRIMARY KEY,
    subnet        VARCHAR(64)  NOT NULL,
    scan_type     VARCHAR(32)  NOT NULL DEFAULT 'discovery',
    status        VARCHAR(20)  NOT NULL DEFAULT 'running'
                      CHECK (status IN ('running', 'completed', 'failed')),
    host_count    INTEGER      NOT NULL DEFAULT 0,
    error_message TEXT,
    started_by    INTEGER      REFERENCES app_users(id) ON DELETE SET NULL,
    started_at    TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finished_at   TIMESTAMP
);
CREATE INDEX IF NOT EXISTS network_scans_started ON network_scans (started_at DESC);

CREATE TABLE IF NOT EXISTS network_scan_hosts (
    id                BIGSERIAL PRIMARY KEY,
    scan_id           INTEGER      NOT NULL REFERENCES network_scans(id) ON DELETE CASCADE,
    ip_address        INET         NOT NULL,
    hostname          VARCHAR(255),
    mac_address       VARCHAR(17),
    vendor            VARCHAR(128),
    device_type       VARCHAR(32)  NOT NULL DEFAULT 'unknown',
    os_guess          VARCHAR(128),
    open_ports        JSONB        NOT NULL DEFAULT '[]',
    is_access_point   BOOLEAN      NOT NULL DEFAULT FALSE,
    is_radius_capable BOOLEAN      NOT NULL DEFAULT FALSE,
    latency_ms        DOUBLE PRECISION DEFAULT 0,
    created_at        TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS network_scan_hosts_scan ON network_scan_hosts (scan_id);
CREATE INDEX IF NOT EXISTS network_scan_hosts_ip   ON network_scan_hosts (ip_address);

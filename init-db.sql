-- ============================================================
-- FreeRADIUS Manager - Database Initialization Script
-- PostgreSQL 15+
-- ============================================================

-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- APPLICATION USERS (Admins, Operators)
-- ============================================================
CREATE TABLE IF NOT EXISTS app_users (
    id               SERIAL PRIMARY KEY,
    username         VARCHAR(50)  UNIQUE NOT NULL,
    password_hash    VARCHAR(255) NOT NULL,
    email            VARCHAR(100) NOT NULL,
    full_name        VARCHAR(100),
    role             VARCHAR(20)  NOT NULL CHECK (role IN ('super_admin', 'admin', 'operator')),
    mfa_secret       VARCHAR(255),
    mfa_enabled      BOOLEAN      DEFAULT FALSE,
    is_active        BOOLEAN      DEFAULT TRUE,
    last_login       TIMESTAMP,
    failed_attempts  INTEGER      DEFAULT 0,
    locked_until     TIMESTAMP,
    created_at       TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- RADIUS USERS (Network access users)
-- ============================================================
CREATE TABLE IF NOT EXISTS radius_users (
    id                    SERIAL PRIMARY KEY,
    username              VARCHAR(64)  UNIQUE NOT NULL,
    password              VARCHAR(255) NOT NULL,
    email                 VARCHAR(100),
    full_name             VARCHAR(100),
    department            VARCHAR(50),
    status                VARCHAR(20)  DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'expired')),
    device_limit          INTEGER      DEFAULT 1 CHECK (device_limit BETWEEN 1 AND 20),
    account_expiry        DATE,
    password_expiry       DATE,
    force_password_change BOOLEAN      DEFAULT FALSE,
    created_by            INTEGER      REFERENCES app_users(id) ON DELETE SET NULL,
    created_at            TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    updated_at            TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
);

-- Password history to prevent reuse
CREATE TABLE IF NOT EXISTS radius_user_password_history (
    id           SERIAL PRIMARY KEY,
    user_id      INTEGER NOT NULL REFERENCES radius_users(id) ON DELETE CASCADE,
    password_hash VARCHAR(255) NOT NULL,
    changed_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- FREERADIUS CORE TABLES (must match FreeRADIUS schema)
-- ============================================================
CREATE TABLE IF NOT EXISTS radcheck (
    id        SERIAL PRIMARY KEY,
    username  VARCHAR(64) NOT NULL DEFAULT '',
    attribute VARCHAR(64) NOT NULL DEFAULT '',
    op        CHAR(2)     NOT NULL DEFAULT ':=',
    value     VARCHAR(253) NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS radcheck_username ON radcheck (username, attribute);

CREATE TABLE IF NOT EXISTS radreply (
    id        SERIAL PRIMARY KEY,
    username  VARCHAR(64) NOT NULL DEFAULT '',
    attribute VARCHAR(64) NOT NULL DEFAULT '',
    op        CHAR(2)     NOT NULL DEFAULT '=',
    value     VARCHAR(253) NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS radreply_username ON radreply (username, attribute);

CREATE TABLE IF NOT EXISTS radgroupcheck (
    id        SERIAL PRIMARY KEY,
    groupname VARCHAR(64) NOT NULL DEFAULT '',
    attribute VARCHAR(64) NOT NULL DEFAULT '',
    op        CHAR(2)     NOT NULL DEFAULT ':=',
    value     VARCHAR(253) NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS radgroupcheck_groupname ON radgroupcheck (groupname, attribute);

CREATE TABLE IF NOT EXISTS radgroupreply (
    id        SERIAL PRIMARY KEY,
    groupname VARCHAR(64) NOT NULL DEFAULT '',
    attribute VARCHAR(64) NOT NULL DEFAULT '',
    op        CHAR(2)     NOT NULL DEFAULT '=',
    value     VARCHAR(253) NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS radgroupreply_groupname ON radgroupreply (groupname, attribute);

CREATE TABLE IF NOT EXISTS radusergroup (
    id        SERIAL PRIMARY KEY,
    username  VARCHAR(64) NOT NULL DEFAULT '',
    groupname VARCHAR(64) NOT NULL DEFAULT '',
    priority  INTEGER     NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS radusergroup_username ON radusergroup (username);

-- ============================================================
-- NAS CLIENTS
-- ============================================================
CREATE TABLE IF NOT EXISTS nas (
    id          SERIAL PRIMARY KEY,
    nasname     VARCHAR(128) NOT NULL UNIQUE,
    shortname   VARCHAR(32),
    type        VARCHAR(30)  DEFAULT 'other',
    ports       INTEGER,
    secret      VARCHAR(128) NOT NULL,
    server      VARCHAR(64),
    community   VARCHAR(50),
    description TEXT,
    status      VARCHAR(20)  DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at  TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- RADIUS ACCOUNTING
-- ============================================================
CREATE TABLE IF NOT EXISTS radacct (
    radacctid          BIGSERIAL    PRIMARY KEY,
    acctsessionid      VARCHAR(64)  NOT NULL DEFAULT '',
    acctuniqueid       VARCHAR(32)  NOT NULL DEFAULT '',
    username           VARCHAR(64)  NOT NULL DEFAULT '',
    realm              VARCHAR(64)  DEFAULT '',
    nasipaddress       INET         NOT NULL,
    nasportid          VARCHAR(15),
    nasporttype        VARCHAR(32),
    acctstarttime      TIMESTAMP,
    acctupdatetime     TIMESTAMP,
    acctstoptime       TIMESTAMP,
    acctinterval       INTEGER,
    acctsessiontime    BIGINT       DEFAULT 0,
    acctauthentic      VARCHAR(32),
    connectinfo_start  VARCHAR(50),
    connectinfo_stop   VARCHAR(50),
    acctinputoctets    BIGINT       DEFAULT 0,
    acctoutputoctets   BIGINT       DEFAULT 0,
    calledstationid    VARCHAR(50)  NOT NULL DEFAULT '',
    callingstationid   VARCHAR(50)  NOT NULL DEFAULT '',
    acctterminatecause VARCHAR(32)  NOT NULL DEFAULT '',
    servicetype        VARCHAR(32),
    framedprotocol     VARCHAR(32),
    framedipaddress    INET,
    framedipv6address  INET,
    framedipv6prefix   INET,
    framedinterfaceid  VARCHAR(44),
    delegatedipv6prefix INET,
    class              VARCHAR(64)
);

CREATE INDEX IF NOT EXISTS radacct_username     ON radacct (username);
CREATE INDEX IF NOT EXISTS radacct_starttime    ON radacct (acctstarttime);
CREATE INDEX IF NOT EXISTS radacct_stoptime     ON radacct (acctstoptime);
CREATE INDEX IF NOT EXISTS radacct_nasip        ON radacct (nasipaddress);
CREATE INDEX IF NOT EXISTS radacct_callingstationid ON radacct (callingstationid);
CREATE UNIQUE INDEX IF NOT EXISTS radacct_uniqid ON radacct (acctuniqueid);

-- ============================================================
-- AUDIT LOG
-- ============================================================
CREATE TABLE IF NOT EXISTS audit_log (
    id          BIGSERIAL PRIMARY KEY,
    user_id     INTEGER REFERENCES app_users(id) ON DELETE SET NULL,
    action      VARCHAR(50)  NOT NULL,
    target_type VARCHAR(50),
    target_id   INTEGER,
    details     JSONB,
    ip_address  INET,
    user_agent  TEXT,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS audit_log_user_id   ON audit_log (user_id);
CREATE INDEX IF NOT EXISTS audit_log_action    ON audit_log (action);
CREATE INDEX IF NOT EXISTS audit_log_created   ON audit_log (created_at DESC);

-- Post-auth logging (required by FreeRADIUS queries.conf)
CREATE TABLE IF NOT EXISTS radpostauth (
    id               BIGSERIAL PRIMARY KEY,
    username         VARCHAR(64)  NOT NULL DEFAULT '',
    pass             VARCHAR(64)  NOT NULL DEFAULT '',
    reply            VARCHAR(32)  NOT NULL DEFAULT '',
    calledstationid  VARCHAR(50)  DEFAULT '',
    callingstationid VARCHAR(50)  DEFAULT '',
    authdate         TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    nasipaddress     INET
);
CREATE INDEX IF NOT EXISTS radpostauth_username ON radpostauth (username);
CREATE INDEX IF NOT EXISTS radpostauth_authdate  ON radpostauth (authdate);

-- ============================================================
-- SYSTEM SETTINGS
-- ============================================================
CREATE TABLE IF NOT EXISTS system_settings (
    key         VARCHAR(100) PRIMARY KEY,
    value       TEXT,
    description TEXT,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- REFRESH TOKENS
-- ============================================================
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          SERIAL PRIMARY KEY,
    user_id     INTEGER NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    token_hash  VARCHAR(255) NOT NULL UNIQUE,
    expires_at  TIMESTAMP NOT NULL,
    revoked     BOOLEAN DEFAULT FALSE,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS refresh_tokens_user ON refresh_tokens (user_id);
CREATE INDEX IF NOT EXISTS refresh_tokens_hash ON refresh_tokens (token_hash);

-- ============================================================
-- TRIGGERS - auto-update updated_at
-- ============================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_app_users_updated_at
    BEFORE UPDATE ON app_users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_radius_users_updated_at
    BEFORE UPDATE ON radius_users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_nas_updated_at
    BEFORE UPDATE ON nas
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- DEFAULT DATA
-- ============================================================

-- Default super admin (password: Admin@123456 - CHANGE IMMEDIATELY)
-- bcrypt hash of 'Admin@123456' with cost 12
INSERT INTO app_users (username, password_hash, email, full_name, role)
VALUES (
    'superadmin',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj/VK3RQK5cm',
    'admin@localhost.local',
    'Super Administrator',
    'super_admin'
) ON CONFLICT (username) DO NOTHING;

-- Default system settings
INSERT INTO system_settings (key, value, description) VALUES
    ('smtp_host',           '',          'SMTP server hostname'),
    ('smtp_port',           '587',       'SMTP server port'),
    ('smtp_user',           '',          'SMTP username'),
    ('smtp_from',           'noreply@radius-manager.local', 'From email address'),
    ('password_min_length', '12',        'Minimum password length'),
    ('password_expiry_days','90',        'Default password expiry in days'),
    ('session_timeout',     '3600',      'Session timeout in seconds'),
    ('mfa_required',        'false',     'Require MFA for all admin users'),
    ('backup_schedule',     '0 2 * * *', 'Cron schedule for automated backups'),
    ('backup_retention',    '30',        'Backup retention in days'),
    ('max_device_limit',    '20',        'Maximum devices per user'),
    ('rate_limit_per_min',  '100',       'API rate limit per IP per minute'),
    ('brute_force_attempts','5',         'Failed login attempts before lockout'),
    ('brute_force_lockout', '15',        'Lockout duration in minutes')
ON CONFLICT (key) DO NOTHING;

-- Default NAS entry (localhost for testing)
INSERT INTO nas (nasname, shortname, type, secret, description)
VALUES ('127.0.0.1', 'localhost', 'other', 'testing123', 'Local test client')
ON CONFLICT (nasname) DO NOTHING;

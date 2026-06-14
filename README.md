# RADIUS Manager

A complete, production-ready network authentication platform built on **FreeRADIUS 3.2**, **PostgreSQL**, **Go**, and **Vue.js 3**. Manage RADIUS users, NAS devices, monitor sessions, and run a first-time setup wizard — all from a clean web interface.

---

## Table of Contents

1. [Features](#features)
2. [Architecture](#architecture)
3. [Requirements](#requirements)
4. [Installation & First-Time Setup](#installation--first-time-setup)
5. [Environment Variables](#environment-variables)
6. [User Roles & Permissions](#user-roles--permissions)
7. [Configuring Network Devices](#configuring-network-devices)
8. [API Reference](#api-reference)
9. [Operations & Maintenance](#operations--maintenance)
10. [Security Hardening](#security-hardening)
11. [Troubleshooting](#troubleshooting)
12. [FAQ](#faq)

---

## Features

| Category | What it does |
|---|---|
| **First-Run Wizard** | 6-step GUI collects org name, admin account, RADIUS secret, security policies — setup page locks forever after first use |
| **RADIUS Authentication** | FreeRADIUS 3.2.10 backed by PostgreSQL, supports PAP, CHAP, MS-CHAP, EAP/PEAP |
| **User Management** | Create, edit, suspend, activate, delete RADIUS users; per-user device limit (1–500) |
| **Bulk Operations** | CSV import/export of RADIUS users with password generation |
| **NAS Devices** | Add/edit/delete NAS clients; test connectivity from the UI; auto-discover local devices |
| **Session Monitoring** | View active sessions, per-user session history, forced disconnect |
| **Auth & Audit Logs** | Every login attempt, every admin action is logged with IP and timestamp |
| **RBAC** | Three roles: Super Admin, Admin, Operator — all enforced server-side |
| **JWT Security** | 15-minute access tokens + 7-day refresh tokens with rotation |
| **MFA** | TOTP-based two-factor authentication (Google Authenticator compatible) |
| **Rate Limiting** | Per-IP brute-force protection with configurable lockout |
| **Backup / Restore** | One-click database backup and restore from the UI |
| **Docker** | Single `docker-compose up -d` starts everything |

---

## Architecture

```
┌────────────────────────────────────────────────────────────┐
│                    Docker Network (bridge)                  │
│                                                            │
│   Browser                                                  │
│      │  :8081                                             │
│      ▼                                                     │
│  ┌───────────┐   /api/v1/*   ┌──────────────────────────┐ │
│  │ Frontend  │ ────────────▶ │  Backend  (Go + Gin)      │ │
│  │ Vue.js 3  │               │  REST API  :8088          │ │
│  │ Nginx :80 │               └──────────┬───────────────┘ │
│  └───────────┘                          │                  │
│                                         │                  │
│                              ┌──────────┴──────────┐      │
│                              │   PostgreSQL :5432   │      │
│                              └──────────┬──────────┘      │
│                                         │                  │
│                              ┌──────────┴──────────┐      │
│  NAS Devices ──────────────▶ │  FreeRADIUS 3.2.10  │      │
│  (Routers, APs, Switches)    │  Auth  :1812/UDP    │      │
│                              │  Acct  :1813/UDP    │      │
│                              └─────────────────────┘      │
└────────────────────────────────────────────────────────────┘
```

### Services

| Container | Image | Purpose |
|---|---|---|
| `radius_postgres` | postgres:15-alpine | Database for app users, RADIUS users, sessions, NAS clients |
| `radius_freeradius` | freeradius/freeradius-server:3.2.10 | RADIUS auth & accounting server |
| `radius_backend` | Go 1.21 (built locally) | REST API, business logic, JWT auth |
| `radius_frontend` | Node 20 + Nginx (built locally) | Web UI served by Nginx, proxies `/api/` to backend |

---

## Requirements

| Requirement | Minimum |
|---|---|
| OS | Linux (Ubuntu 20.04+, Debian 11+, Rocky 8+) |
| Docker | 20.10+ |
| Docker Compose | v2.0+ (plugin) or v1.29+ (standalone) |
| RAM | 1 GB free |
| Disk | 2 GB free |
| Open ports | `8081/TCP` (web UI), `1812/UDP` (RADIUS auth), `1813/UDP` (RADIUS accounting) |

> **Windows / macOS:** Use Docker Desktop. The stack works identically.

---

## Installation & First-Time Setup

### Step 1 — Clone the project

```bash
git clone https://github.com/your-org/radius-manager.git
cd radius-manager
```

Or upload the project folder to your server and `cd` into it.

---

### Step 2 — Create your `.env` file

```bash
cp .env.example .env
```

Open `.env` and set **at minimum** these three values:

```bash
# Strong random string — used to sign JWT tokens
JWT_SECRET=replace-with-64-or-more-random-characters

# Database password
DB_PASSWORD=a-strong-database-password

# RADIUS shared secret — must match what you configure on every NAS device
RADIUS_SECRET=your-radius-shared-secret-min-16-chars
```

> Generate strong secrets quickly:
> ```bash
> openssl rand -hex 32   # for JWT_SECRET
> openssl rand -hex 16   # for RADIUS_SECRET
> ```

---

### Step 3 — Build and start all containers

```bash
docker-compose up -d --build
```

This will:
- Pull the FreeRADIUS and PostgreSQL base images
- Build the Go backend and Vue.js frontend
- Start all four services in the correct order
- Run the database schema initialisation automatically

First build takes **2–4 minutes**. Subsequent starts take under 10 seconds.

---

### Step 4 — Complete the Setup Wizard

Open your browser and go to:

```
http://YOUR_SERVER_IP:8081
```

You will be automatically redirected to the **Setup Wizard**. Fill in each step:

| Step | What you configure |
|---|---|
| 1 — Welcome | Overview of what gets configured |
| 2 — Organisation | Organisation name and timezone |
| 3 — Admin Account | Super-admin username, email, full name, password |
| 4 — RADIUS Settings | RADIUS shared secret (must match your `.env` `RADIUS_SECRET`) |
| 5 — Security Policies | Password policy, session timeout, brute-force lockout |
| 6 — Review & Finish | Confirm all settings and submit |

After clicking **Complete Setup**:
- The system creates your admin account
- The setup page is **permanently locked** (returns 404 for all future requests)
- You are automatically redirected to the login page

> **Important:** The setup page can only be completed **once**. If you need to reset and start over, follow the [Factory Reset](#factory-reset) procedure below.

---

### Step 5 — Log in

Use the admin credentials you set in the wizard.

| Field | Value |
|---|---|
| URL | `http://YOUR_SERVER_IP:8081` |
| Username | The username you chose in the wizard |
| Password | The password you chose in the wizard |

---

### Verify everything is healthy

```bash
# All containers should show "Up"
docker-compose ps

# Check FreeRADIUS is accepting RADIUS requests
SECRET=$(docker exec radius_freeradius sh -c 'echo $RADIUS_SECRET')
docker exec radius_freeradius radtest YOUR_USERNAME YOUR_PASSWORD 127.0.0.1 1812 "$SECRET"
# Expected: "Received Access-Accept"

# Check the backend API
curl http://localhost:8088/api/v1/health
# Expected: {"status":"ok","services":{"database":"ok"}}
```

---

## Environment Variables

Full list of all supported variables in `.env`:

### Database

| Variable | Default | Description |
|---|---|---|
| `DB_HOST` | `postgres` | PostgreSQL hostname (Docker service name) |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `radius_user` | Database username |
| `DB_PASSWORD` | — | **Required.** Database password |
| `DB_NAME` | `radius` | Database name |
| `DB_SSL_MODE` | `disable` | Set to `require` in production behind TLS |

### JWT Authentication

| Variable | Default | Description |
|---|---|---|
| `JWT_SECRET` | — | **Required.** Secret for signing tokens. Use 64+ random chars |
| `JWT_ACCESS_EXPIRY` | `15m` | Access token lifetime |
| `JWT_REFRESH_EXPIRY` | `168h` | Refresh token lifetime (7 days) |

### RADIUS

| Variable | Default | Description |
|---|---|---|
| `RADIUS_HOST` | `freeradius` | FreeRADIUS hostname (Docker service name) |
| `RADIUS_AUTH_PORT` | `1812` | Authentication port |
| `RADIUS_ACCT_PORT` | `1813` | Accounting port |
| `RADIUS_SECRET` | — | **Required.** Shared secret. Must match every NAS device config |
| `RADIUS_TIMEOUT` | `5s` | Timeout for backend→FreeRADIUS packets |
| `RADIUS_RETRIES` | `3` | Retry count for test packets |

### Web Interface

| Variable | Default | Description |
|---|---|---|
| `WEB_PORT` | `8088` | Backend API internal port |
| `VITE_API_URL` | `http://localhost:8088` | API base URL used by the frontend during build |

### Security

| Variable | Default | Description |
|---|---|---|
| `ADMIN_IP_WHITELIST` | `0.0.0.0/0` | CIDR to restrict admin access. Example: `10.0.0.0/8` |
| `SESSION_TIMEOUT` | `3600` | Idle session timeout in seconds |
| `BCRYPT_COST` | `12` | bcrypt cost factor for password hashing (10–14 recommended) |
| `RATE_LIMIT` | `100` | Max requests per minute per IP |
| `BRUTE_FORCE_ATTEMPTS` | `5` | Failed logins before lockout |
| `BRUTE_FORCE_LOCKOUT` | `15` | Lockout duration in minutes |

### Email (Optional)

| Variable | Default | Description |
|---|---|---|
| `SMTP_HOST` | — | SMTP server hostname |
| `SMTP_PORT` | `587` | SMTP port |
| `SMTP_USER` | — | SMTP username |
| `SMTP_PASS` | — | SMTP password |
| `SMTP_FROM` | `noreply@radius-manager.local` | From address |

### Backup

| Variable | Default | Description |
|---|---|---|
| `BACKUP_SCHEDULE` | `0 2 * * *` | Cron schedule for automatic backups (2 AM daily) |
| `BACKUP_RETENTION_DAYS` | `30` | Days to keep automatic backups |

---

## User Roles & Permissions

| Permission | Super Admin | Admin | Operator |
|---|:---:|:---:|:---:|
| View RADIUS users | ✅ | ✅ | ✅ |
| Create RADIUS users | ✅ | ✅ | ❌ |
| Edit RADIUS users | ✅ | ✅ | ❌ |
| Delete RADIUS users | ✅ | ✅ | ❌ |
| Reset user password | ✅ | ✅ | ✅ |
| Suspend / activate user | ✅ | ✅ | ❌ |
| Force disconnect user | ✅ | ✅ | ❌ |
| Bulk import / export CSV | ✅ | ✅ | ❌ |
| View NAS devices | ✅ | ✅ | ✅ |
| Add / edit / delete NAS | ✅ | ✅ | ❌ |
| Test NAS connectivity | ✅ | ✅ | ❌ |
| View active sessions | ✅ | ✅ | ✅ |
| View auth logs | ✅ | ✅ | ✅ |
| View audit logs | ✅ | ✅ | ❌ |
| Manage admin users | ✅ | ❌ | ❌ |
| System settings | ✅ | ❌ | ❌ |
| Backup / restore | ✅ | ❌ | ❌ |

---

## Configuring Network Devices

All NAS devices must use the **same `RADIUS_SECRET`** you set in `.env` and the wizard.

### MikroTik RouterOS (Hotspot / PPPoE)

```routeros
/radius
add address=YOUR_SERVER_IP secret=YOUR_RADIUS_SECRET service=hotspot,ppp

/ip hotspot profile
set [find default=yes] use-radius=yes

/ppp profile
set [find default=yes] use-radius=yes
```

### MikroTik — Add Accounting

```routeros
/radius
set [find address=YOUR_SERVER_IP] accounting-port=1813

/ip hotspot profile
set [find] accounting=yes
```

### Cisco IOS / IOS-XE

```
aaa new-model
radius server RADIUS-MGR
 address ipv4 YOUR_SERVER_IP auth-port 1812 acct-port 1813
 key YOUR_RADIUS_SECRET

aaa group server radius RADIUS-GROUP
 server name RADIUS-MGR

aaa authentication login default group RADIUS-GROUP local
aaa accounting network default start-stop group RADIUS-GROUP
```

### Cisco WLC (Wireless LAN Controller)

Go to **Security → AAA → RADIUS → Authentication** and add:
- Server IP: `YOUR_SERVER_IP`
- Port: `1812`
- Shared Secret: `YOUR_RADIUS_SECRET`

Then under **Security → AAA → RADIUS → Accounting**, add the same server with port `1813`.

### Ubiquiti UniFi

Go to **Settings → Profiles → RADIUS** and click **Create New**:
- RADIUS Auth Server: `YOUR_SERVER_IP` port `1812`
- RADIUS Accounting Server: `YOUR_SERVER_IP` port `1813`
- Shared Secret: `YOUR_RADIUS_SECRET`

Assign the profile to a wireless network under **Settings → WiFi**.

### pfSense / OPNsense (Captive Portal)

Navigate to **Services → Captive Portal → Edit Zone → RADIUS**:
- Primary Authentication Server: `YOUR_SERVER_IP`
- Authentication Port: `1812`
- Accounting Port: `1813`
- Shared Secret: `YOUR_RADIUS_SECRET`

### Huawei / H3C / TP-Link Omada

```
radius-server authentication YOUR_SERVER_IP 1812 key YOUR_RADIUS_SECRET
radius-server accounting YOUR_SERVER_IP 1813 key YOUR_RADIUS_SECRET
aaa authentication-scheme radius
aaa accounting-scheme radius
```

---

## API Reference

Base URL: `http://YOUR_SERVER:8088/api/v1`

All endpoints except `/auth/login`, `/setup/status`, and `/health` require:
```
Authorization: Bearer <access_token>
```

### Health

```
GET  /health
GET  /api/v1/health
```
Returns `{"status":"ok","services":{"database":"ok"}}` when healthy.

### Setup Wizard (first run only)

```
GET  /api/v1/setup/status     — check if setup is required
POST /api/v1/setup/complete   — submit wizard form
```
Both endpoints return **404** once setup has been completed.

### Authentication

| Method | Path | Description |
|---|---|---|
| POST | `/auth/login` | Login → returns `access_token` + `refresh_token` |
| POST | `/auth/refresh` | Exchange refresh token for new access token |
| POST | `/auth/logout` | Revoke refresh token |
| POST | `/auth/change-password` | Change own password |
| POST | `/auth/mfa/setup` | Get TOTP QR code |
| POST | `/auth/mfa/verify` | Confirm TOTP and enable MFA |

### RADIUS Users

| Method | Path | Min Role | Description |
|---|---|---|---|
| GET | `/radius/users` | Operator | List all RADIUS users (paginated) |
| POST | `/radius/users` | Admin | Create a new RADIUS user |
| GET | `/radius/users/:id` | Operator | Get user details |
| PUT | `/radius/users/:id` | Admin | Update user |
| DELETE | `/radius/users/:id` | Admin | Delete user |
| POST | `/radius/users/:id/reset-password` | Operator | Reset user password |
| POST | `/radius/users/:id/suspend` | Admin | Suspend user account |
| POST | `/radius/users/:id/activate` | Admin | Activate suspended account |
| POST | `/radius/users/:id/disconnect` | Admin | Force disconnect active sessions |
| GET | `/radius/users/:id/sessions` | Operator | User session history |
| POST | `/radius/users/import` | Admin | Bulk import via CSV |
| GET | `/radius/users/export` | Admin | Export all users to CSV |

### NAS Devices

| Method | Path | Min Role | Description |
|---|---|---|---|
| GET | `/nas` | Operator | List all NAS devices |
| POST | `/nas` | Admin | Add a NAS device |
| GET | `/nas/:id` | Operator | Get NAS details |
| PUT | `/nas/:id` | Admin | Update NAS device |
| DELETE | `/nas/:id` | Admin | Remove NAS device |
| POST | `/nas/:id/test` | Admin | Test RADIUS connectivity |
| POST | `/nas/discover` | Admin | Auto-discover NAS on local subnet |

### Sessions & Logs

| Method | Path | Min Role | Description |
|---|---|---|---|
| GET | `/sessions/active` | Operator | All active sessions |
| GET | `/sessions/user/:username` | Operator | Sessions for a specific user |
| GET | `/logs/auth` | Operator | Authentication logs |
| GET | `/logs/audit` | Admin | Admin action audit trail |

### Statistics

| Method | Path | Min Role | Description |
|---|---|---|---|
| GET | `/statistics/dashboard` | Operator | Dashboard stats (users, sessions, top NAS) |

### System (Super Admin only)

| Method | Path | Description |
|---|---|---|
| GET | `/settings` | Get system settings |
| PUT | `/settings` | Update system settings |
| POST | `/backup` | Create database backup |
| GET | `/backups` | List available backups |
| POST | `/restore` | Restore from backup |

---

## Operations & Maintenance

### View live logs

```bash
# All services
docker-compose logs -f

# Individual service
docker-compose logs -f freeradius
docker-compose logs -f backend
docker-compose logs -f frontend
docker-compose logs -f postgres
```

### Restart a service

```bash
docker-compose restart freeradius
docker-compose restart backend
```

### Stop and start everything

```bash
docker-compose down        # stop, remove containers (data is safe in volumes)
docker-compose up -d       # start again
```

### Manual database backup

```bash
# Backup
docker exec radius_postgres pg_dump -U radius_user radius \
  > backup_$(date +%Y%m%d_%H%M%S).sql

# Restore
cat backup_20240101_120000.sql | \
  docker exec -i radius_postgres psql -U radius_user radius
```

### Test RADIUS authentication manually

```bash
# Get the current shared secret from the container
SECRET=$(docker exec radius_freeradius sh -c 'echo $RADIUS_SECRET')

# Test a specific user
docker exec radius_freeradius radtest USERNAME PASSWORD 127.0.0.1 1812 "$SECRET"
# Success: "Received Access-Accept"
# Failure: "Received Access-Reject"
```

### Update to a new version

```bash
git pull
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

### Factory Reset

Wipes all data and allows the setup wizard to run again:

```bash
docker-compose down
docker volume rm free_postgres_data free_radius_logs
docker-compose up -d --build
```

> **Warning:** This deletes all users, sessions, NAS devices, and logs permanently.

---

## Security Hardening

### Before going to production, complete this checklist:

- [ ] **Change all secrets** — `DB_PASSWORD`, `JWT_SECRET`, `RADIUS_SECRET` in `.env`
- [ ] **Use strong RADIUS secret** — minimum 20 characters, no dictionary words
- [ ] **Enable HTTPS** — put Nginx or Traefik in front with Let's Encrypt TLS
- [ ] **Restrict admin access** — set `ADMIN_IP_WHITELIST=10.0.0.0/8` (your management network)
- [ ] **Enable MFA** — go to profile settings and scan the QR code with Google Authenticator
- [ ] **Firewall rules** — only expose ports `8081/TCP`, `1812/UDP`, `1813/UDP` to the internet
- [ ] **Review NAS clients** — remove the default "localhost" NAS entry if not needed
- [ ] **Set session timeout** — reduce `SESSION_TIMEOUT` to `1800` (30 min) for sensitive environments
- [ ] **Configure email** — set SMTP variables for password reset and alert notifications

### Recommended firewall rules (ufw example)

```bash
sudo ufw allow 8081/tcp      # web UI — restrict to your office IP in production
sudo ufw allow 1812/udp      # RADIUS auth — only from NAS device IPs
sudo ufw allow 1813/udp      # RADIUS accounting — only from NAS device IPs
sudo ufw deny 8088/tcp       # backend API — never expose directly, only via nginx
sudo ufw deny 5432/tcp       # PostgreSQL — internal only
```

### HTTPS with Nginx reverse proxy

```nginx
server {
    listen 443 ssl;
    server_name radius.yourdomain.com;

    ssl_certificate     /etc/letsencrypt/live/radius.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/radius.yourdomain.com/privkey.pem;

    location / {
        proxy_pass http://localhost:8081;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

server {
    listen 80;
    server_name radius.yourdomain.com;
    return 301 https://$host$request_uri;
}
```

---

## Troubleshooting

### Web UI not loading

```bash
# Check all containers are running
docker-compose ps

# Check frontend logs
docker-compose logs frontend
```

Common causes:
- Port `8081` already in use → change `ports: - "8081:80"` in `docker-compose.yml`
- Frontend build failed → run `docker-compose build --no-cache frontend`

---

### Cannot log in after setup

```bash
# Verify the backend is up
curl http://localhost:8088/api/v1/health

# Check backend logs for auth errors
docker-compose logs backend | grep -i error
```

Common causes:
- Wrong password → check what you entered in the wizard
- Backend not running → `docker-compose restart backend`

---

### RADIUS: "Access-Reject" for a user

```bash
# 1. Check the user exists in radcheck
docker exec radius_postgres psql -U radius_user -d radius \
  -c "SELECT * FROM radcheck WHERE username='USERNAME';"

# 2. Test directly from inside the FreeRADIUS container
SECRET=$(docker exec radius_freeradius sh -c 'echo $RADIUS_SECRET')
docker exec radius_freeradius radtest USERNAME PASSWORD 127.0.0.1 1812 "$SECRET"

# 3. Check FreeRADIUS auth log
docker logs radius_freeradius 2>&1 | grep USERNAME | tail -10
```

Common causes:
- User doesn't exist → create via web UI
- Wrong password → reset via web UI
- User suspended → activate via web UI
- `Cleartext-Password` stored with wrong attribute name → check radcheck table

---

### RADIUS: "Unknown client" error

```bash
docker logs radius_freeradius 2>&1 | grep "unknown client"
```

Means the NAS device IP is not in `clients.conf`. Fix:
1. Add the NAS via the web UI under **NAS Devices**
2. Or verify the NAS IP is within the configured CIDR ranges (`172.x.x.x/16`, `10.x.x.x/8`, `192.168.x.x/16`)

---

### RADIUS: "Shared secret is incorrect"

The NAS device is using a different shared secret than what is configured in RADIUS Manager. Fix:
1. Check `RADIUS_SECRET` in your `.env`
2. Re-run `docker-compose down && docker-compose up -d` so FreeRADIUS picks up the new value
3. Update the shared secret on the NAS device to match

---

### RADIUS: SQL "Ignoring" warning at startup

```
Warning: Ignoring "sql" (see raddb/mods-available/README.rst)
```

This means FreeRADIUS cannot load the SQL module. Run:

```bash
# Check the actual mods-available/sql inside the container
docker exec radius_freeradius cat /etc/freeradius/mods-available/sql

# Check DB credentials are correctly substituted
docker exec radius_freeradius grep -E "server|login|password" /etc/freeradius/mods-available/sql

# Force a full rebuild
docker-compose down
docker-compose build --no-cache freeradius
docker-compose up -d
```

---

### Database connection error in backend

```bash
# Check PostgreSQL is healthy
docker-compose ps postgres

# Check backend environment
docker exec radius_backend env | grep DB_

# Verify connection manually
docker exec radius_postgres psql -U radius_user -d radius -c "SELECT 1;"
```

---

### Port already in use

```bash
# See what is using the port
sudo ss -tulnp | grep -E ':8081|:1812|:1813|:8088'

# Stop conflicting containers from previous runs
docker-compose down --remove-orphans
docker-compose up -d
```

---

## FAQ

**Q: Can I use this with an existing FreeRADIUS installation?**
No — this stack manages its own FreeRADIUS container. You can export the user data from RADIUS Manager and import it into an external FreeRADIUS.

**Q: Does it support EAP-TLS (certificate-based auth)?**
FreeRADIUS 3.2 supports EAP-TLS. The default configuration enables PEAP and EAP-TTLS out of the box. For EAP-TLS you need to generate and install client certificates in the FreeRADIUS `certs/` directory.

**Q: How do I add more supported Docker bridge subnets?**
Edit `freeradius/clients.conf`, add a new `client` block for your subnet, and run:
```bash
docker-compose build --no-cache freeradius && docker-compose up -d freeradius
```

**Q: The setup wizard ran but I want to change the admin password.**
Log in and go to your **Profile → Change Password**, or use the admin user management screen if you are a super admin.

**Q: Can I run multiple instances for different ISP zones?**
Yes — deploy separate stacks in separate directories (or with different `COMPOSE_PROJECT_NAME` values) on different ports.

**Q: How do I completely wipe all data and start over?**
```bash
docker-compose down
docker volume rm free_postgres_data free_radius_logs
docker-compose up -d --build
```

---

## License

MIT — Free to use, modify, and deploy in production.

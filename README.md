# FreeRADIUS Manager Tool

<div align="center">

![Version](https://img.shields.io/badge/version-2.0.0--pro-blue)
![Go](https://img.shields.io/badge/Go-1.21-00ADD8?logo=go)
![Vue](https://img.shields.io/badge/Vue.js-3-4FC08D?logo=vue.js)
![FreeRADIUS](https://img.shields.io/badge/FreeRADIUS-3.2-red)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-336791?logo=postgresql)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker)
![License](https://img.shields.io/badge/license-MIT-green)

**A complete open-source FreeRADIUS management platform.**  
Multi-tenant · Billing · CRM · Security Suite · Captive Portal · Webhooks · and much more.

[Live Demo](#) · [Report Bug](https://github.com/phinias955/FreeRADIUS-Manager-Tool/issues) · [Request Feature](https://github.com/phinias955/FreeRADIUS-Manager-Tool/issues)

</div>

---

## Table of Contents

1. [Overview](#overview)
2. [Feature Matrix](#feature-matrix)
3. [Architecture](#architecture)
4. [Requirements](#requirements)
5. [Quick Start](#quick-start)
6. [First-Time Setup Wizard](#first-time-setup-wizard)
7. [Environment Variables](#environment-variables)
8. [User Roles & Permissions](#user-roles--permissions)
9. [Module Documentation](#module-documentation)
   - [RADIUS Users](#radius-users)
   - [NAS Devices](#nas-devices)
   - [User Plans & Billing](#user-plans--billing)
   - [Vouchers](#vouchers)
   - [Bandwidth Profiles](#bandwidth-profiles)
   - [IP Pools](#ip-pools)
   - [Hotspot Zones](#hotspot-zones)
   - [Captive Portal Builder](#captive-portal-builder)
   - [User Self-Service Portal](#user-self-service-portal)
   - [Organizations (Multi-Tenancy)](#organizations-multi-tenancy)
   - [CRM — Customers & Tickets](#crm--customers--tickets)
   - [Payments](#payments)
   - [Promotions & Discount Codes](#promotions--discount-codes)
   - [RADIUS Templates](#radius-templates)
   - [Bulk Operations](#bulk-operations)
   - [SMS Notifications](#sms-notifications)
   - [Email Alerts](#email-alerts)
   - [Webhooks](#webhooks)
   - [Scheduler](#scheduler)
   - [API Keys](#api-keys)
   - [Reports & Analytics](#reports--analytics)
   - [Live Stats (SSE)](#live-stats-sse)
   - [Network Map](#network-map)
   - [Security Suite (Tier 7)](#security-suite-tier-7)
10. [API Reference](#api-reference)
11. [Connecting Network Devices](#connecting-network-devices)
12. [Operations & Maintenance](#operations--maintenance)
13. [Upgrading](#upgrading)
14. [Troubleshooting](#troubleshooting)
15. [FAQ](#faq)
16. [Contributing](#contributing)
17. [License](#license)

---

## Overview

**FreeRADIUS Manager Tool** is a production-ready, all-in-one web platform that sits on top of FreeRADIUS and PostgreSQL. It replaces the need for manual `radtest`, editing `radcheck`/`radreply` tables by hand, and complex shell scripts — with a clean, role-based web interface built for ISPs, enterprises, hotspot operators, and network engineers.

### What it solves

| Problem | Solution |
|---|---|
| Managing hundreds of RADIUS users manually | Web UI with bulk operations, CSV import/export, templates |
| No visibility into who is online | Live dashboard with session monitoring and kick |
| Complex billing for ISP plans | User Plans, Invoices, Payments with receipt generation |
| Security blind spots | Honeypot listener, credential stuffing detection, GeoIP enforcement |
| Multi-site management | Hotspot Zones, NAS grouping, Network Map |
| No customer portal | Self-service portal for end-users |
| Integrations with external systems | Webhooks with HMAC signing, REST API with API keys |

---

## Feature Matrix

### Core Platform
| Feature | Description |
|---|---|
| First-Run Setup Wizard | 6-step GUI; page locks permanently after first use |
| RADIUS Authentication | PAP, CHAP, MS-CHAP, EAP/PEAP via FreeRADIUS 3.2 |
| User Management | Create, edit, suspend, activate, delete RADIUS users |
| NAS Device Management | Add, test, monitor NAS devices with live ping status |
| Session Monitoring | View active sessions, kick users, view history |
| Role-Based Access Control | Super Admin, Admin, Operator — scoped permissions |
| MFA / 2FA | TOTP-based second factor for admin accounts |
| JWT Authentication | Short-lived access tokens + refresh token rotation |
| Dark Mode | Full dark/light theme toggle |

### Tier 1 — Monetisation Basics
| Feature | Description |
|---|---|
| Vouchers | Generate, disable, export voucher codes for prepaid access |
| Bandwidth Profiles | Named profiles (e.g. "10Mbps/5Mbps") applied via Mikrotik-Rate-Limit |
| Reports & Analytics | Usage reports, daily trends, NAS usage, auth success rates |

### Tier 2 — Business Layer
| Feature | Description |
|---|---|
| User Plans | Define plans with speed, quota, expiry; assign to users |
| Billing / Invoices | Auto-generate invoices, track payment status |
| NAS Monitor | Background ping worker; live latency badge on NAS list |
| Email Alerts | Configurable alert rules sent via SMTP |

### Tier 3 — Advanced System
| Feature | Description |
|---|---|
| IP Pools | CIDR-based pools, auto-assign/release static IPs to users |
| API Keys | Generate bearer keys for external system integration |
| Scheduler | Built-in task runner (expire users, cleanup, custom scripts) |
| CSV Import | Advanced multi-column bulk user import with validation |

### Tier 4 — Network Management
| Feature | Description |
|---|---|
| Hotspot Zones | Group NAS devices into logical zones with stats |
| Network Map | Visual NAS topology with live online/offline status |
| User Self-Service Portal | End-users log in with RADIUS credentials to view usage |
| Live Stats (SSE) | Real-time dashboard counters via Server-Sent Events |
| SMS Notifications | Send individual or bulk SMS via configurable gateway |

### Tier 5 — Enterprise
| Feature | Description |
|---|---|
| Organizations / Resellers | Multi-tenant support; isolate users, NAS, invoices per org |
| CRM — Customers | Full customer profiles with contracts and notes |
| Support Tickets | Built-in helpdesk ticketing linked to customers |
| Captive Portal Builder | Brand and deploy custom login pages per zone |
| Webhooks | Event-driven HTTP callbacks with HMAC-SHA256 signing |

### Tier 6 — Financial & Operations
| Feature | Description |
|---|---|
| Payments | Record payments, link to invoices, generate HTML receipts |
| Promotions / Discounts | Discount codes with usage limits, date ranges, percentage/fixed |
| RADIUS Templates | Reusable attribute sets applied to any number of users |
| Bulk Operations | Suspend, activate, delete, change plan, set expiry — mass actions |

### Tier 7 — Security Suite
| Feature | Description |
|---|---|
| RADIUS Simulator | Send real UDP Access-Request packets, inspect full attribute exchange |
| GeoIP Enforcement | Block/flag/allow countries; live IP lookup with VPN detection |
| Honeypot Listener | Decoy RADIUS on UDP 11812; logs all probes, raises alerts |
| Credential Stuffing Detection | Sliding-window per-IP rate limiter; auto-blocks repeat offenders |
| Pattern Analysis | Background goroutine scans `radpostauth` every 5 min for attacks |
| Security Alerts | Severity-graded feed (low/medium/high/critical) with acknowledge |
| IP Block Manager | Manual + automatic block list with expiry and reason |

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     Browser / Client                     │
└─────────────────────┬───────────────────────────────────┘
                      │ HTTPS (port 8081)
┌─────────────────────▼───────────────────────────────────┐
│              Vue.js 3 Frontend (Nginx)                   │
│  Composition API · Pinia · Vue Router · Tailwind CSS    │
└─────────────────────┬───────────────────────────────────┘
                      │ REST API + SSE (port 8088)
┌─────────────────────▼───────────────────────────────────┐
│              Go Backend (Gin Framework)                  │
│  JWT Auth · RBAC · Background Workers · Webhooks        │
└──────┬──────────────────────────────┬───────────────────┘
       │ SQL                          │ UDP 1812/1813
┌──────▼──────────┐         ┌─────────▼───────────────────┐
│   PostgreSQL 15 │         │     FreeRADIUS 3.2           │
│  radcheck       │◄────────│  rlm_sql_postgresql          │
│  radreply       │         │  PAP · CHAP · MS-CHAP · EAP  │
│  radacct        │         └─────────────────────────────┘
│  radpostauth    │
│  app tables     │         ┌─────────────────────────────┐
└─────────────────┘         │  Honeypot (UDP 11812)        │
                            │  Go goroutine (built-in)     │
                            └─────────────────────────────┘
```

All four services run as Docker containers and communicate over an internal bridge network.

---

## Requirements

| Requirement | Minimum | Recommended |
|---|---|---|
| OS | Ubuntu 20.04 / Debian 11 | Ubuntu 22.04 LTS |
| RAM | 1 GB | 4 GB |
| CPU | 1 core | 2+ cores |
| Disk | 10 GB | 50 GB |
| Docker | 20.10+ | latest |
| Docker Compose | 1.29+ | v2 |
| Open ports | 1812/udp, 1813/udp, 8081/tcp | + 11812/udp (honeypot) |

---

## Quick Start

```bash
# 1. Clone the repository
git clone https://github.com/phinias955/FreeRADIUS-Manager-Tool.git
cd FreeRADIUS-Manager-Tool

# 2. Copy and edit environment file
cp .env.example .env
nano .env        # Set DB_PASSWORD, JWT_SECRET, RADIUS_SECRET at minimum

# 3. Start all services
docker-compose up -d

# 4. Check all containers are running
docker-compose ps
```

Then open **http://your-server-ip:8081** — the setup wizard will appear automatically.

> **Tip:** To follow logs: `docker-compose logs -f backend`

---

## First-Time Setup Wizard

The setup wizard runs **once only**. After completion it is permanently locked (returns 404) to prevent re-configuration by attackers.

### Step-by-step

| Step | What you configure |
|---|---|
| 1. Welcome | Review requirements |
| 2. Database | Verify PostgreSQL connectivity |
| 3. Organization | Company name, contact email, timezone |
| 4. Admin Account | Username, full name, strong password |
| 5. RADIUS Settings | Shared secret (must match NAS devices), server host |
| 6. Security | Session timeout, brute-force lockout, IP whitelist |

After completing all steps, the wizard redirects to the login page and the `/api/v1/setup/*` routes become permanently inaccessible.

### Security notes
- Setup endpoint is rate-limited to 10 requests per IP
- The setup completion state is cached in memory — a restart does not re-open it
- All setup input is validated server-side before writing to the database

---

## Environment Variables

Copy `.env.example` to `.env` and set every value before first run.

### Required

| Variable | Example | Description |
|---|---|---|
| `DB_PASSWORD` | `Str0ngP@ss!` | PostgreSQL password |
| `JWT_SECRET` | `64-char-random-string` | Signs all JWT tokens — keep secret |
| `RADIUS_SECRET` | `32-char-minimum` | Shared secret between server and NAS devices |

### Database

| Variable | Default | Description |
|---|---|---|
| `DB_HOST` | `postgres` | PostgreSQL host (Docker service name) |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `radius_user` | Database username |
| `DB_NAME` | `radius` | Database name |
| `DB_SSL_MODE` | `disable` | `disable` / `require` / `verify-full` |
| `DB_MAX_OPEN_CONNS` | `25` | Connection pool max open |
| `DB_MAX_IDLE_CONNS` | `5` | Connection pool max idle |

### Application

| Variable | Default | Description |
|---|---|---|
| `WEB_PORT` | `8081` | Frontend port exposed to the internet |
| `APP_ENV` | `production` | `production` disables Gin debug output |
| `VITE_API_URL` | `http://localhost:8088` | URL the browser uses to reach the API |

### JWT

| Variable | Default | Description |
|---|---|---|
| `JWT_ACCESS_EXPIRY` | `15m` | Access token lifetime |
| `JWT_REFRESH_EXPIRY` | `168h` | Refresh token lifetime (7 days) |

### Email (SMTP)

| Variable | Example | Description |
|---|---|---|
| `SMTP_HOST` | `smtp.gmail.com` | SMTP server hostname |
| `SMTP_PORT` | `587` | SMTP port (587 = STARTTLS) |
| `SMTP_USER` | `admin@example.com` | SMTP login username |
| `SMTP_PASS` | `app-password` | SMTP password / app password |
| `SMTP_FROM` | `noreply@example.com` | From address for system emails |

### SMS Gateway

| Variable | Example | Description |
|---|---|---|
| `SMS_GATEWAY` | `https://api.smsprovider.com/send` | HTTP SMS endpoint |
| `SMS_API_KEY` | `your-api-key` | API key for SMS gateway |
| `SMS_SENDER_ID` | `NetManager` | Sender name/number |
| `SMS_BODY_TEMPLATE` | `Hi {name}, your account expires {date}` | Message template |

### Security (Tier 7)

| Variable | Default | Description |
|---|---|---|
| `HONEYPOT_PORT` | `11812` | UDP port for the honeypot RADIUS listener |

### Credential Stuffing Thresholds (set in Admin → Settings)

| Setting Key | Default | Description |
|---|---|---|
| `cs_max_fails` | `10` | Failed auth attempts before blocking |
| `cs_window_secs` | `300` | Sliding window in seconds |
| `cs_block_mins` | `60` | Block duration in minutes |
| `honeypot_enabled` | `true` | Enable/disable honeypot listener |
| `geoip_mode` | `flag` | `flag` / `block` / `disabled` |

---

## User Roles & Permissions

| Permission | Super Admin | Admin | Operator |
|---|:---:|:---:|:---:|
| Setup wizard | ✅ | ❌ | ❌ |
| Manage admin users | ✅ | ❌ | ❌ |
| System settings | ✅ | ❌ | ❌ |
| Organizations | ✅ | ✅ | ❌ |
| RADIUS users (create/edit/delete) | ✅ | ✅ | ❌ |
| RADIUS users (view/search) | ✅ | ✅ | ✅ |
| NAS devices | ✅ | ✅ | view |
| Plans & Billing | ✅ | ✅ | view |
| Payments | ✅ | ✅ | create |
| Bulk Operations | ✅ | ✅ | ❌ |
| Vouchers | ✅ | ✅ | view |
| Reports | ✅ | ✅ | ✅ |
| Security Center (view) | ✅ | ✅ | ✅ |
| Security Center (block IP, rules) | ✅ | ✅ | ❌ |
| RADIUS Simulator | ✅ | ✅ | ❌ |
| API Keys | ✅ | ✅ | ❌ |
| Webhooks | ✅ | ✅ | ❌ |

---

## Module Documentation

### RADIUS Users

**Path:** `/users`

Create and manage FreeRADIUS authentication accounts. Each user maps directly to rows in the `radcheck` and `radreply` tables.

**Key attributes set automatically:**
- `Cleartext-Password` — stores the login password
- `Simultaneous-Use` — maximum concurrent sessions (1–500)
- `Expiration` — account expiry date (`Jan 01 2026 00:00:00`)
- `Mikrotik-Rate-Limit` — upload/download speed limit from bandwidth profile
- `Framed-IP-Address` — static IP from an IP pool (optional)

**CSV Import format:**
```
username,password,expiry,max_sessions,bandwidth_profile
alice,Pass1234,2027-01-01,2,10mbps
bob,Secure99!,2026-06-30,1,
```

---

### NAS Devices

**Path:** `/nas`

Register all network devices (routers, access points, switches) that send RADIUS requests.

| Field | Description |
|---|---|
| NAS Name | IP address or hostname |
| Short Name | Display label |
| Type | `cisco`, `mikrotik`, `other`, etc. |
| Ports | Number of ports (informational) |
| Shared Secret | Must match the device configuration **exactly** |
| Zone | Assign to a Hotspot Zone |

**RADIUS Test** — sends a live `Access-Request` UDP packet to verify connectivity. Shows latency in milliseconds.

**Disconnect (CoA)** — sends a `Disconnect-Request` to terminate a specific user session remotely (requires NAS CoA support).

---

### User Plans & Billing

**Path:** `/plans` · `/billing`

Define internet service plans and assign them to users.

**Plan fields:**
- Name, description, price
- Speed (upload/download Mbps)
- Data quota (GB, 0 = unlimited)
- Duration (days)
- Validity period

**Invoice lifecycle:** `pending` → `paid` → `overdue`

Invoices are generated automatically when a plan is assigned. The billing page shows all invoices with filter by status, user, and date range.

---

### Vouchers

**Path:** `/vouchers`

Generate pre-paid access codes for time-limited or data-limited access.

- Generate batches of 1–1000 vouchers
- Each voucher has a unique code, expiry, and usage limit
- Export to CSV for printing
- Disable individual vouchers

---

### Bandwidth Profiles

**Path:** `/bandwidth`

Named speed profiles applied to users via the `Mikrotik-Rate-Limit` RADIUS attribute.

**Format example:** `10M/5M` (10 Mbps down / 5 Mbps up)

Profiles can be applied to individual users or via bulk operations.

---

### IP Pools

**Path:** `/ip-pools`

Define CIDR ranges and auto-assign static IP addresses to RADIUS users.

```
Pool: "ISP-Main"  CIDR: 192.168.100.0/24
→ 254 available IPs auto-managed
→ Assigned IPs written to radreply as Framed-IP-Address
```

**Operations:** assign, release, view all assignments.

---

### Hotspot Zones

**Path:** `/zones`

Group NAS devices into logical sites (e.g. "Downtown", "Airport", "School").

Each zone shows:
- Number of NAS devices
- Currently active users
- Zone-level statistics

---

### Captive Portal Builder

**Path:** `/captive-portal`

Design and deploy custom branded login pages for hotspot zones without writing HTML.

**Configurable fields:**
- Logo URL
- Background color / image
- Title, welcome message, footer text
- Primary button color

The portal is served at a public URL: `/api/v1/captive/serve/:id`

---

### User Self-Service Portal

**Path:** `/portal` (public, no admin login required)

End-users authenticate with their **RADIUS credentials** and see:
- Account status and expiry
- Current data usage (upload/download)
- Active sessions
- Session history
- Assigned IP address

> Accessible at `http://your-server:8081/portal`

---

### Organizations (Multi-Tenancy)

**Path:** `/organizations`

Create tenant accounts for resellers or branch offices. Each organization has its own:
- RADIUS users
- NAS devices
- Invoices and revenue tracking
- Admin users (scoped to that org)

---

### CRM — Customers & Tickets

**Path:** `/customers` · `/tickets`

**Customers** — full profiles separate from RADIUS users:
- Contact info, contract type, contract value
- Notes and history
- Link to RADIUS username

**Tickets** — internal helpdesk:
- Open, in-progress, closed, resolved statuses
- Priority levels (low, medium, high, urgent)
- Full description and resolution notes
- Linked to customer and assigned admin

---

### Payments

**Path:** `/payments`

Record financial transactions against invoices.

- Payment methods: cash, bank transfer, mobile money, card, other
- Automatic invoice status update to `paid` on payment record
- HTML receipt generation (printable)
- Revenue summary dashboard (today, this month, total)
- Webhook event `invoice.paid` fired on every payment

---

### Promotions & Discount Codes

**Path:** `/promotions`

Create discount codes for plan promotions.

| Field | Description |
|---|---|
| Code | Unique promo code (e.g. `SAVE20`) |
| Type | `percentage` or `fixed` |
| Value | Discount amount |
| Max Uses | Total uses allowed (0 = unlimited) |
| Valid From / Until | Date range |
| Active | Toggle on/off |

**Validate endpoint:** `POST /api/v1/promotions/validate` — returns discount amount and final price.

---

### RADIUS Templates

**Path:** `/templates`

Save reusable sets of `radcheck`/`radreply` attributes as named templates.

**Use cases:**
- "VIP Package" — high speed + unlimited data
- "Expired Account" — set all users to expired state
- "School Network" — content filter attributes

Apply a template to any number of users in one click. Clone templates to create variations.

---

### Bulk Operations

**Path:** `/bulk`

Perform mass actions on multiple RADIUS users simultaneously.

| Action | Description |
|---|---|
| `suspend` | Set Simultaneous-Use to 0 |
| `activate` | Restore Simultaneous-Use |
| `delete` | Remove users and all their RADIUS attributes |
| `change_plan` | Assign a new plan to all selected users |
| `set_expiry` | Set expiry date for all selected users |
| `apply_template` | Apply a RADIUS template to all selected users |
| `reset_attributes` | Clear all radcheck/radreply entries |

All bulk operations are logged to the `bulk_operations` history table.

---

### SMS Notifications

**Path:** `/sms`

Send text messages via any HTTP SMS gateway.

- Send individual SMS to a user's phone number
- Bulk expiry notification (all users expiring in N days)
- SMS log with delivery status
- Template variables: `{name}`, `{expiry}`, `{username}`

**Supported gateways:** Any HTTP/REST gateway (Africa's Talking, Twilio, BulkSMS, etc.)

---

### Email Alerts

**Path:** `/alerts`

Define rules that trigger email notifications when conditions are met.

| Condition | Example trigger |
|---|---|
| Failed auth rate | > 50 failures per hour |
| NAS offline | Device unreachable for > 5 minutes |
| User quota | User exceeds 80% of data quota |

Emails are sent via the configured SMTP server.

---

### Webhooks

**Path:** `/webhooks`

Register external HTTP endpoints to receive real-time event notifications.

**Available events:**
- `user.created` · `user.deleted` · `user.suspended`
- `invoice.paid` · `invoice.created`
- `nas.offline` · `nas.online`
- `session.started` · `session.stopped`

**Security:** Every delivery is signed with `X-Signature: sha256=<hmac>` using your webhook secret. Verify it server-side to authenticate the payload.

**Retry:** Failed deliveries are logged to `webhook_logs` with HTTP status and response body.

---

### Scheduler

**Path:** `/scheduler`

Built-in background task runner. Configure recurring jobs:

| Task | Default Schedule | Description |
|---|---|---|
| Expire Users | Daily 00:05 | Suspends users past their expiry date |
| Cleanup Sessions | Every 6 hours | Removes stale open sessions |
| Invoice Overdue | Daily 00:10 | Marks unpaid invoices older than 30 days as overdue |
| GeoIP Cache Cleanup | Daily 02:00 | Removes stale GeoIP lookups |

Tasks can be enabled/disabled, their schedule changed (cron syntax), and run manually.

---

### API Keys

**Path:** `/api-keys`

Generate bearer tokens for external systems to authenticate with the API without using admin credentials.

```http
GET /api/v1/radius/users
Authorization: ApiKey frm_xxxxxxxxxxxxxxxx
```

Keys can be:
- Named and scoped
- Enabled/disabled without deletion
- Rotated (delete old + create new)

---

### Reports & Analytics

**Path:** `/reports`

| Report | Description |
|---|---|
| Usage Summary | Total auth attempts, accepts, rejects, sessions |
| Daily Trend | Auth success/failure counts per day (30 days) |
| NAS Usage | Per-device session counts and data transferred |
| Top Users | Highest data consumers |
| Auth History | Full `radpostauth` log with filter |
| Export | Download reports as CSV |

---

### Live Stats (SSE)

The dashboard receives real-time updates every 5 seconds via **Server-Sent Events** without page refresh.

**Live counters:**
- Total active sessions
- Auth attempts (last 60 seconds)
- Online NAS devices
- Failed auth rate

The SSE stream is authenticated via `?token=<jwt>` query parameter (EventSource does not support custom headers).

---

### Network Map

**Path:** `/network-map`

Visual representation of all NAS devices showing:
- Online / offline status (color coded)
- Active user count per device
- Ping latency
- Zone grouping

---

### Security Suite (Tier 7)

#### Security Center
**Path:** `/security`

Unified security dashboard showing:
- Unread high/critical alert count
- Active blocked IPs
- Honeypot probes today
- Failed auth rate (last hour)
- Credential stuffing patterns detected
- 24-hour auth failure bar chart
- Alert feed with severity filter and acknowledge

#### RADIUS Simulator
**Path:** `/security/simulator`

Send real `Access-Request` UDP packets to FreeRADIUS and inspect every detail of the exchange:
- Full request attribute list (User-Name, NAS-IP, Service-Type, etc.)
- Full reply attribute list (Reply-Message, Framed-IP-Address, etc.)
- Raw packet hex dumps for both request and reply
- Latency measurement
- Batch mode: test up to 20 username/password pairs at once

**Use cases:** verify a user's credentials work, debug attribute delivery, test NAS connectivity.

#### GeoIP Enforcement
Looks up the country, city, ISP, and VPN status of any IP address using ip-api.com (no API key required, results cached for 24 hours).

**Rule actions:**
- `block` — flag for immediate access denial
- `flag` — raise a security alert but allow access
- `allow` — whitelist (useful to override a broad block rule)

#### Honeypot Listener
A silent decoy RADIUS server running on UDP port 11812. Legitimate clients never contact it — only scanners and attackers do.

- Logs every probe: source IP, username attempted, NAS IP, raw attributes
- Raises a `high`-severity security alert on first contact from a new IP
- Responds with `Access-Reject` to slow down attackers without revealing it's a honeypot

**Enable/disable:** via `honeypot_enabled` setting in the database.

#### Credential Stuffing Detection

**Realtime (sliding window):**
- Tracks failed authentication attempts per source IP in memory
- When `cs_max_fails` failures occur within `cs_window_secs` seconds, the IP is automatically blocked for `cs_block_mins` minutes
- A `critical` security alert is raised immediately
- Block records persist in the database and survive restarts

**Pattern Analysis (background):**
- Runs every 5 minutes
- Scans `radpostauth` for IPs that tried 5+ different usernames in 10 minutes (distributed credential stuffing)
- Raises `high`-severity alerts per attacker IP

---

## API Reference

All API endpoints are under `/api/v1/`. Authentication is via `Authorization: Bearer <jwt>` header or `ApiKey <key>` for API key auth.

### Authentication

| Method | Path | Description |
|---|---|---|
| `POST` | `/auth/login` | Login with username + password, returns JWT |
| `POST` | `/auth/refresh` | Exchange refresh token for new access token |
| `POST` | `/auth/logout` | Invalidate current session |
| `POST` | `/auth/change-password` | Change own password |

### RADIUS Users

| Method | Path | Description |
|---|---|---|
| `GET` | `/radius/users` | List users (paginated, searchable) |
| `POST` | `/radius/users` | Create user |
| `GET` | `/radius/users/:id` | Get single user |
| `PUT` | `/radius/users/:id` | Update user |
| `DELETE` | `/radius/users/:id` | Delete user |
| `POST` | `/radius/users/import` | Bulk import from CSV |
| `GET` | `/radius/users/export` | Export all users as CSV |
| `POST` | `/radius/users/:id/disconnect` | Send CoA Disconnect-Request |

### NAS Devices

| Method | Path | Description |
|---|---|---|
| `GET` | `/nas` | List NAS devices |
| `POST` | `/nas` | Add NAS device |
| `PUT` | `/nas/:id` | Update NAS device |
| `DELETE` | `/nas/:id` | Delete NAS device |
| `POST` | `/nas/:id/test` | Test RADIUS connectivity |

### Security (Tier 7)

| Method | Path | Description |
|---|---|---|
| `GET` | `/security/summary` | Security health snapshot |
| `GET` | `/security/alerts` | List alerts (filter by severity/type) |
| `PUT` | `/security/alerts/:id/ack` | Acknowledge alert |
| `PUT` | `/security/alerts/ack-all` | Acknowledge all alerts |
| `DELETE` | `/security/alerts/:id` | Delete alert |
| `GET` | `/security/blocked-ips` | List blocked IPs |
| `POST` | `/security/blocked-ips` | Manually block an IP |
| `DELETE` | `/security/blocked-ips/:id` | Unblock an IP |
| `GET` | `/security/geoip/lookup?ip=x.x.x.x` | GeoIP lookup |
| `GET` | `/security/geoip/rules` | List country rules |
| `POST` | `/security/geoip/rules` | Add country rule |
| `DELETE` | `/security/geoip/rules/:id` | Delete country rule |
| `GET` | `/security/honeypot/status` | Honeypot status |
| `GET` | `/security/honeypot/logs` | Honeypot probe logs |
| `DELETE` | `/security/honeypot/logs` | Clear old honeypot logs |
| `POST` | `/radius/simulate` | Simulate RADIUS auth (single) |
| `POST` | `/radius/simulate/batch` | Simulate RADIUS auth (batch) |

---

## Connecting Network Devices

### MikroTik RouterOS

```routeros
/radius
add address=<SERVER_IP> secret=<RADIUS_SECRET> service=login,hotspot timeout=3s

/ip hotspot profile
set [find] use-radius=yes

/ip hotspot user profile
set [find] use-radius=yes
```

### Cisco IOS

```ios
aaa new-model
aaa authentication login default group radius local
aaa authorization exec default group radius local

radius server RADIUS-SRV
 address ipv4 <SERVER_IP> auth-port 1812 acct-port 1813
 key <RADIUS_SECRET>
```

### Ubiquiti UniFi

1. Settings → Profiles → RADIUS → Create New
2. IP: `<SERVER_IP>`, Port: `1812`, Secret: `<RADIUS_SECRET>`
3. Assign profile to SSID

### OpenWRT / Hostapd

```
/etc/config/wireless:
option auth_server <SERVER_IP>
option auth_port 1812
option auth_secret <RADIUS_SECRET>
option acct_server <SERVER_IP>
option acct_port 1813
option acct_secret <RADIUS_SECRET>
```

> See [DEVICE-INTEGRATION.md](./DEVICE-INTEGRATION.md) for full step-by-step guides for 15+ device types.

---

## Operations & Maintenance

### View container status
```bash
docker-compose ps
docker-compose logs -f backend
docker-compose logs -f freeradius
```

### Restart a service
```bash
docker-compose restart backend
docker-compose restart freeradius
```

### Database backup
```bash
docker exec radius_postgres pg_dump -U radius_user radius > backup_$(date +%Y%m%d).sql
```

### Database restore
```bash
docker exec -i radius_postgres psql -U radius_user radius < backup_20260101.sql
```

### Re-initialise the database schema (⚠ destroys all data)
```bash
docker exec -i radius_postgres psql -U radius_user radius < init-db.sql
```

### Scale horizontally
The backend is stateless — run multiple replicas behind a load balancer. The PostgreSQL database is the single source of truth.

---

## Upgrading

```bash
# Pull latest code
git pull origin main

# Rebuild and restart services
docker-compose build backend frontend
docker-compose up -d

# Apply any new database migrations
docker exec -i radius_postgres psql -U radius_user radius < init-db.sql
```

> Schema migrations use `CREATE TABLE IF NOT EXISTS` and `INSERT ... ON CONFLICT DO NOTHING` — safe to re-run.

---

## Troubleshooting

### Login fails with "Internal Server Error"
```bash
docker logs radius_backend | tail -50
# Check DB_PASSWORD in .env matches the postgres container
```

### RADIUS test shows "Unreachable"
1. Verify `RADIUS_HOST` in `.env` is `freeradius` (not `localhost`)
2. Check FreeRADIUS is running: `docker-compose ps freeradius`
3. Check the shared secret matches both `.env` and the NAS device config

### Users can't authenticate from NAS
1. Go to NAS Devices → Test — confirm "Access-Accept" response
2. Check the NAS shared secret matches the one in the database
3. Run RADIUS Simulator with the exact username/password to isolate the issue

### Honeypot won't start
- Ensure port 11812/udp is not already in use: `ss -ulnp | grep 11812`
- Check `honeypot_enabled` setting in Admin → Settings is `true`

### "Failed to configure RADIUS attributes" on user create
- Password must be at least 6 characters
- Username must be unique (not already in `radcheck`)

### Frontend shows "System Error" badge
```bash
curl http://localhost:8088/api/v1/health
# Should return {"status":"ok","services":{"database":"ok"}}
```
If database is "error", check PostgreSQL is running and credentials are correct.

---

## FAQ

**Q: Can I use this without Docker?**  
A: Yes. Install Go 1.21+, PostgreSQL 15, and FreeRADIUS 3.2 manually. Set all environment variables and run `go build -o radius-manager ./...` inside the `backend/` folder.

**Q: Does it support LDAP or Active Directory?**  
A: Not natively. FreeRADIUS supports LDAP via `rlm_ldap` — configure it in the FreeRADIUS config files and the manager will still track sessions and accounting.

**Q: Is the data encrypted at rest?**  
A: RADIUS passwords are stored as `Cleartext-Password` in PostgreSQL by default (required by PAP/CHAP). For EAP/PEAP with MS-CHAP, use `NT-Password`. Encrypt the database volume using OS-level encryption (LUKS) for at-rest protection.

**Q: How many users can it handle?**  
A: PostgreSQL handles millions of rows comfortably. FreeRADIUS is designed for high-throughput auth. Performance depends on hardware; 10,000+ concurrent sessions is achievable on modest hardware.

**Q: Can I white-label this?**  
A: Yes. Change the app name in `frontend/src/components/layout/AppLayout.vue` and update the logo URL in the `.env` file.

**Q: How do I reset the admin password?**  
```bash
# Generate a new bcrypt hash
docker exec radius_backend sh -c 'echo -n "NewPassword123!" | htpasswd -niB x | cut -d: -f2'

# Update the database
docker exec radius_postgres psql -U radius_user radius \
  -c "UPDATE app_users SET password_hash='\$2y\$...' WHERE username='admin';"
```

**Q: Can multiple admins use it simultaneously?**  
A: Yes. The backend is fully stateless with JWT authentication. Multiple sessions work independently.

---

## Contributing

Contributions are welcome.

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Commit your changes: `git commit -m "feat: add my feature"`
4. Push: `git push origin feat/my-feature`
5. Open a Pull Request

Please follow the existing code style (Go: `gofmt`, Vue: Composition API + `<script setup>`).

---

## License

This project is licensed under the **MIT License** — see [LICENSE](LICENSE) for details.

---

<div align="center">
Built with ❤ for network engineers and ISPs worldwide.<br>
<a href="https://github.com/phinias955/FreeRADIUS-Manager-Tool">github.com/phinias955/FreeRADIUS-Manager-Tool</a>
</div>

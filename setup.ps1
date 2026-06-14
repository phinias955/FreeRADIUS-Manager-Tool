# FreeRADIUS Manager - Windows Setup Script
# Run as Administrator: PowerShell -ExecutionPolicy Bypass -File setup.ps1

Write-Host "=================================================" -ForegroundColor Blue
Write-Host "   FreeRADIUS Manager - Windows Setup Script     " -ForegroundColor Blue
Write-Host "=================================================" -ForegroundColor Blue
Write-Host ""

# Check Docker
try {
    docker --version | Out-Null
    Write-Host "[OK] Docker found" -ForegroundColor Green
} catch {
    Write-Host "[ERROR] Docker Desktop not found. Install from https://www.docker.com/products/docker-desktop" -ForegroundColor Red
    exit 1
}

# Check Docker is running
try {
    docker info 2>$null | Out-Null
    Write-Host "[OK] Docker daemon is running" -ForegroundColor Green
} catch {
    Write-Host "[ERROR] Docker is not running. Start Docker Desktop first." -ForegroundColor Red
    exit 1
}

# Generate passwords
function New-RandomPassword {
    param([int]$Length = 40)
    $chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
    -join (1..$Length | ForEach-Object { $chars[(Get-Random -Maximum $chars.Length)] })
}

function New-RandomHex {
    param([int]$Bytes = 32)
    -join (1..$Bytes | ForEach-Object { '{0:X2}' -f (Get-Random -Maximum 256) })
}

if (Test-Path ".env") {
    Write-Host "[WARN] .env already exists, skipping password generation" -ForegroundColor Yellow
} else {
    Write-Host "[INFO] Generating secure passwords..." -ForegroundColor Cyan

    $DbPass = New-RandomPassword
    $JwtSecret = New-RandomHex
    $RadiusSecret = New-RandomPassword 32

    $envContent = @"
# FreeRADIUS Manager - Auto-generated Configuration
# Generated on: $(Get-Date)

DB_HOST=postgres
DB_PORT=5432
DB_USER=radius_user
DB_PASSWORD=$DbPass
DB_NAME=radius
DB_SSL_MODE=disable

JWT_SECRET=$JwtSecret
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

RADIUS_HOST=freeradius
RADIUS_AUTH_PORT=1812
RADIUS_ACCT_PORT=1813
RADIUS_SECRET=$RadiusSecret
RADIUS_TIMEOUT=5s
RADIUS_RETRIES=3

WEB_PORT=8080
VITE_API_URL=/api/v1

ADMIN_IP_WHITELIST=0.0.0.0/0
SESSION_TIMEOUT=3600
BCRYPT_COST=12
RATE_LIMIT=100
BRUTE_FORCE_ATTEMPTS=5
BRUTE_FORCE_LOCKOUT=15

APP_ENV=production

SMTP_HOST=
SMTP_PORT=587
SMTP_USER=
SMTP_PASS=
SMTP_FROM=noreply@radius-manager.local

BACKUP_SCHEDULE=0 2 * * *
BACKUP_RETENTION_DAYS=30
"@

    $envContent | Out-File -FilePath ".env" -Encoding utf8
    Write-Host "[OK] .env file created" -ForegroundColor Green
}

# Build and start
Write-Host ""
Write-Host "[INFO] Building Docker images (3-10 minutes)..." -ForegroundColor Cyan
docker-compose build --parallel

Write-Host ""
Write-Host "[INFO] Starting services..." -ForegroundColor Cyan
docker-compose up -d

Write-Host ""
Write-Host "[INFO] Waiting for services (30 seconds)..." -ForegroundColor Cyan
Start-Sleep -Seconds 30

Write-Host ""
Write-Host "=================================================" -ForegroundColor Green
Write-Host "   Setup Complete!                               " -ForegroundColor Green
Write-Host "=================================================" -ForegroundColor Green
Write-Host ""
Write-Host "  Web Interface:  http://localhost:8081" -ForegroundColor Cyan
Write-Host "  Backend API:    http://localhost:8080" -ForegroundColor Cyan
Write-Host "  RADIUS Port:    localhost:1812/UDP" -ForegroundColor Cyan
Write-Host ""
Write-Host "  Default Login:" -ForegroundColor Yellow
Write-Host "  Username: superadmin"
Write-Host "  Password: Admin@123456"
Write-Host ""
Write-Host "  CHANGE THE DEFAULT PASSWORD IMMEDIATELY!" -ForegroundColor Red
Write-Host ""

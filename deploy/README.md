# Navisha Spark — Deployment Guide

## Overview

Panduan deployment Navisha Spark ke VPS dengan Docker, Nginx reverse proxy, dan SSL Let's Encrypt.

## Arsitektur

```
Internet
    │
    ▼
┌──────────────────────────────────────┐
│  DNS: navisha.cloud → 202.155.13.11  │
│  DNS: spark.navisha.cloud → same IP  │
└──────────────────────────────────────┘
    │
    ▼
┌──────────────────────────────────────┐
│  VPS: 202.155.13.11                  │
│  ┌────────────────────────────────┐  │
│  │  Nginx (port 80/443)           │  │
│  │  - SSL termination             │  │
│  │  - Reverse proxy               │  │
│  └──────────┬─────────────────────┘  │
│             │                        │
│  ┌──────────▼─────────────────────┐  │
│  │  Docker Network                  │  │
│  │  ┌───────────────────────────┐  │  │
│  │  │  navisha-spark:8080       │  │  │
│  │  └───────────────────────────┘  │  │
│  │  ┌───────────────────────────┐  │  │
│  │  │  navisha-nginx:80/443     │  │  │
│  │  └───────────────────────────┘  │  │
│  │  ┌───────────────────────────┐  │  │
│  │  │  navisha-certbot          │  │  │
│  │  └───────────────────────────┘  │  │
│  └──────────────────────────────────┘  │
└──────────────────────────────────────┘
```

## Prerequisites

- VPS dengan Ubuntu 22.04/24.04
- Domain sudah terdaftar dan DNS pointing ke IP VPS
- SSH access ke VPS

## Quick Start

### Step 1: Setup VPS

```bash
# SSH ke VPS
ssh root@202.155.13.11

# Jalankan setup (install Docker, Nginx, Certbot, firewall, generate SSH key)
chmod +x deploy/vps-setup.sh
./deploy/vps-setup.sh
```

**Setelah setup selesai, script akan menampilkan SSH public key. Kamu perlu:**

1. Copy public key yang ditampilkan
2. Buka https://github.com/settings/keys
3. Klik "New SSH key"
4. Title: `navisha-spark-vps`
5. Paste public key
6. Klik "Add SSH key"

### Step 2: Clone Repository

```bash
# Clone dari GitHub (setelah SSH key ditambahkan)
git clone git@github.com:jhontea/navisha-spark.git
cd navisha-spark
```

### Step 3: Konfigurasi Environment

```bash
# Copy env example
cp .env.example .env

# Edit dengan kredensial asli
nano .env

# Isi:
# TELEGRAM_BOT_TOKEN=your_bot_token
# TELEGRAM_CHAT_ID=your_chat_id
# DATABASE_URL=your_supabase_connection_string
# OPENROUTER_API_KEY=your_openrouter_api_key
```

### Step 4: Build dan Start Aplikasi

```bash
# Build Docker image dan start semua service
docker-compose -f deploy/docker-compose.prod.yml up -d --build

# Cek status
docker-compose -f deploy/docker-compose.prod.yml ps

# Cek logs
docker-compose -f deploy/docker-compose.prod.yml logs -f spark
```

### Step 5: Setup SSL

```bash
# Pastikan DNS sudah pointing ke VPS terlebih dahulu!
chmod +x deploy/ssl/certbot-init.sh
./deploy/ssl/certbot-init.sh
```

### Step 6: Verifikasi

```bash
# Test health check
curl https://spark.navisha.cloud/healthz

# Test trigger (akan mengirim insight ke Telegram)
curl https://spark.navisha.cloud/trigger

# Test root domain
curl https://navisha.cloud
```

## Endpoints

| Endpoint | URL | Akses |
|----------|-----|-------|
| Health Check | `https://spark.navisha.cloud/healthz` | Public |
| Root | `https://spark.navisha.cloud/` | Public |
| Manual Trigger | `https://spark.navisha.cloud/trigger` | Public (rate-limited) |
| Root Domain | `https://navisha.cloud` | Public |

## Menambah Aplikasi Baru

Untuk menambah aplikasi baru dengan port berbeda:

### 1. Tambah service di `deploy/docker-compose.prod.yml`

```yaml
services:
  # ... spark service sudah ada ...

  new-app:
    image: your-app-image:latest
    container_name: new-app
    expose:
      - "3000"
    restart: unless-stopped
    networks:
      - navisha-network
```

### 2. Buat Nginx config baru di `deploy/nginx/app.navisha.cloud`

```yaml
server {
    listen 443 ssl http2;
    server_name app.navisha.cloud;

    # ... SSL config sama seperti spark ...

    location / {
        proxy_pass http://new-app:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 3. Tambah domain ke `deploy/ssl/certbot-init.sh`

```bash
DOMAINS=(
    "navisha.cloud"
    "www.navisha.cloud"
    "spark.navisha.cloud"
    "app.navisha.cloud"  # Tambah ini
)
```

### 4. Restart dan renew SSL

```bash
docker-compose -f deploy/docker-compose.prod.yml up -d
./deploy/ssl/certbot-init.sh
```

## Troubleshooting

### DNS tidak resolve ke IP VPS

```bash
# Cek dari VPS
curl ifconfig.me  # Harusnya 202.155.13.11

# Cek dari komputer lain
nslookup spark.navisha.cloud
```

### Container tidak start

```bash
# Cek logs
docker-compose -f deploy/docker-compose.prod.yml logs spark

# Cek environment variables
docker-compose -f deploy/docker-compose.prod.yml config
```

### SSL certificate error

```bash
# Renew manual
docker-compose -f deploy/docker-compose.prod.yml run --rm certbot renew

# Cek certificate expiry
docker-compose -f deploy/docker-compose.prod.yml run --rm certbot certificates
```

### Trigger returns 502

```bash
# Pastikan spark container running
docker-compose -f deploy/docker-compose.prod.yml ps

# Cek spark logs
docker-compose -f deploy/docker-compose.prod.yml logs spark

# Test langsung ke container
docker exec navisha-spark wget -q --spider http://localhost:8080/healthz
```

## Maintenance

### Update aplikasi

```bash
git pull origin main
docker-compose -f deploy/docker-compose.prod.yml up -d --build
```

### Lihat logs

```bash
# Semua service
docker-compose -f deploy/docker-compose.prod.yml logs -f

# Hanya spark
docker-compose -f deploy/docker-compose.prod.yml logs -f spark

# Hanya nginx
docker-compose -f deploy/docker-compose.prod.yml logs -f nginx
```

### Backup

```bash
# Backup .env dan config
tar -czf backup-$(date +%Y%m%d).tar.gz .env config/ deploy/
```

### Monitoring

```bash
# Resource usage
docker stats

# Container health
docker inspect --format='{{.State.Health.Status}}' navisha-spark
```

## Security Notes

- Firewall hanya membuka port 22, 80, 443
- `/trigger` endpoint ada rate limiting (5 requests/minute)
- SSL dengan Let's Encrypt (auto-renewal)
- Container jalan sebagai non-root user
- Security headers sudah di-set di Nginx

## Support

Jika ada masalah, buka issue di repository atau cek dokumentasi di `docs/SETUP.md`.
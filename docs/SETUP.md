# Navisha Spark — Setup Guide

## Overview

Panduan lengkap untuk setup dan menjalankan Navisha Spark di lingkungan development dan production.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Quick Start](#quick-start)
3. [Configuration Setup (Deployment)](#configuration-setup-deployment)
4. [Development Setup](#development-setup)
5. [Production Deployment](#production-deployment)
6. [Testing](#testing)
7. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Required

- **Go 1.23+** — [Download](https://go.dev/dl/)
- **Docker & Docker Compose** — [Install Docker](https://docs.docker.com/get-docker/)
- **Supabase Account** — [Sign Up](https://supabase.com/)
- **Telegram Bot Token** — [Create Bot](https://t.me/BotFather)
- **OpenRouter API Key** — [Get Key](https://openrouter.ai/)

### Optional

- **Git** — untuk version control
- **Make** — untuk automation (optional)
- **PostgreSQL Client** — untuk database management

---

## Quick Start

```bash
# 1. Clone & masuk direktori
git clone https://github.com/yourusername/navisha-spark.git
cd navisha-spark

# 2. Setup environment variables (WAJIB)
cp .env.example .env
nano .env   # Isi kredensial asli (lihat section 3.1)

# 3. Setup database di Supabase
# Buka https://supabase.com → SQL Editor → paste migrations/001_init.sql → Run

# 4. Start aplikasi
docker-compose up -d

# 5. Cek status
docker-compose ps
curl http://localhost:8080/healthz
docker-compose logs -f spark
```

---

## Configuration Setup (Deployment)

Ada **2 jenis konfigurasi** yang perlu di-setup saat deployment:

### 3.1 Environment Variables — `.env` (WAJIB)

File `.env` berisi secrets/kredensial. File ini **TIDAK boleh di-commit** ke Git.

#### Cara Setup
```bash
# Copy dari template
cp .env.example .env

# Edit dengan editor
nano .env
```

#### Yang Perlu Diisi
```bash
# === WAJIB DIISI ===

# Telegram Bot Token (dari @BotFather)
TELEGRAM_BOT_TOKEN=yourtoken

# Telegram Chat ID (dari getUpdates)
TELEGRAM_CHAT_ID=yourchatid

# Database Supabase (dari Settings → Database → Connection string)
DATABASE_URL=urlsupabase

# OpenRouter API Key (dari openrouter.ai → Keys)
OPENROUTER_API_KEY=yourapikey

# === OPSIONAL (sudah ada default) ===

# Jadwal
SCHEDULE_CRON=0 */3 * * *        # Tiap 3 jam
TIMEZONE=Asia/Jakarta
ACTIVE_HOURS_START=0              # 24 jam penuh
ACTIVE_HOURS_END=23

# Retry
MAX_RETRIES=3
RETRY_DELAYS=1m,5m,15m

# Deduplikasi & Konten
DEDUP_WINDOW_HOURS=24
LEVEL_DISTRIBUTION=beginner:20,intermediate:50,advanced:30
MIN_DAYS_BEFORE_REPEAT=7
```

#### Cara Apply Perubahan
```bash
# Setelah edit .env, restart container
docker-compose restart spark

# Atau jika container belum running
docker-compose up -d
```

#### Verifikasi
```bash
# Cek apakah environment terbaca
docker-compose exec spark env | grep TELEGRAM_BOT_TOKEN
docker-compose exec spark env | grep DATABASE_URL
```

---

### 3.2 Unified YAML Config — `config/config.yaml` (Hot-Reload)

Satu file untuk semua konfigurasi: **topik**, **jadwal**, **level distribution**, **LLM**, **format**, dan **logging**.
Bisa diedit kapan saja **tanpa restart**.

#### Cara Setup Awal
```bash
nano config/config.yaml
```

#### Konfigurasi Topik (categories)
```yaml
categories:
  - name: "Golang"
    enabled: true           # true = aktif, false = skip
    weight: 1.0             # Semakin besar, semakin sering muncul
    subtopics:
      - "concurrency"
      - "goroutine"
      - "channel"
      - "GMP scheduler"

  - name: "Database"
    enabled: true
    weight: 1.0
    subtopics:
      - "indexing"
      - "query planning"
      - "transaction isolation"

  - name: "AI/ML untuk Backend Engineer"
    enabled: false           # Contoh: disable dulu, aktifkan nanti
    weight: 1.0
    subtopics:
      - "dasar LLM"
      - "vector DB"
      - "RAG"
```

#### Konfigurasi Jadwal (schedule)

**Skenario 1: Default (tiap 3 jam, 24 jam)**
```yaml
schedule:
  cron: "0 */3 * * *"        # 00:00, 03:00, 06:00, ..., 21:00
  timezone: "Asia/Jakarta"
  active_hours:
    start: 0                  # Mulai jam 00:00
    end: 23                   # Sampai jam 23:00
```

**Skenario 2: Hanya jam kerja (08:00 - 17:00 WIB)**
```yaml
schedule:
  cron: "0 */3 * * *"        # 08:00, 11:00, 14:00, 17:00
  timezone: "Asia/Jakarta"
  active_hours:
    start: 8
    end: 17
```

**Skenario 3: Lebih jarang (tiap 6 jam)**
```yaml
schedule:
  cron: "0 */6 * * *"        # 00:00, 06:00, 12:00, 18:00
  timezone: "Asia/Jakarta"
  active_hours:
    start: 0
    end: 23
```

#### Cara Apply (HOT-RELOAD — tanpa restart!)
```bash
# Cukup edit file, perubahan langsung berlaku
nano config/config.yaml
```

#### Verifikasi
```bash
# Cek log untuk konfirmasi reload
docker-compose logs spark | grep -i "config reloaded"
```

---

### 3.3 Port & Network — `docker-compose.override.yml` (Perlu Restart)

Jika port `8080` sudah dipakai aplikasi lain, buat file override:

```bash
nano docker-compose.override.yml
```

```yaml
services:
  spark:
    ports:
      - "9090:8080"   # Host port 9090, container port 8080
```

```bash
# Apply perubahan
docker-compose up -d
```

---

### 3.4 Flow Lengkap Setup Config Pertama Kali

```bash
# 1. Clone project
git clone <repo-url>
cd navisha-spark

# 2. Setup environment (WAJIB)
cp .env.example .env
nano .env
# Isi: TELEGRAM_BOT_TOKEN, TELEGRAM_CHAT_ID, DATABASE_URL, OPENROUTER_API_KEY

# 3. (Opsional) Sesuaikan topik atau jadwal di satu file
nano config/config.yaml
# Contoh: disable "AI/ML", ganti active_hours ke 8-17

# 4. Setup database di Supabase
# Buka https://supabase.com → SQL Editor → paste isi migrations/001_init.sql → Run

# 5. Start aplikasi
docker-compose up -d

# 6. Verifikasi semua beres
docker-compose ps                     # Status running
curl http://localhost:8080/healthz    # Response: {"status":"ok",...}
docker-compose logs spark | tail -20  # Cek startup logs
```

---

### 3.5 Cara Update Config Saat Aplikasi Sudah Running

| Yang Mau Diubah | File | Cara | Restart? |
|----------------|------|------|:--------:|
| Ganti bot token | `.env` | `nano .env` → `docker-compose restart spark` | ✅ Ya |
| Ganti DB URL | `.env` | `nano .env` → `docker-compose restart spark` | ✅ Ya |
| Tambah/hapus topik | `config/config.yaml` | `nano config/config.yaml` | ❌ Tidak |
| Ganti jadwal/active hours | `config/config.yaml` | `nano config/config.yaml` | ❌ Tidak |
| Ganti level distribution | `config/config.yaml` | `nano config/config.yaml` | ❌ Tidak |
| Ganti port | `docker-compose.override.yml` | `nano docker-compose.override.yml` → `docker-compose up -d` | ✅ Ya |
| Rollback config | `git checkout config/config.yaml` | Langsung hot-reload | ❌ Tidak |

---

## Development Setup

### 1. Install Go Dependencies

```bash
go mod download
```

### 2. Setup Database (Local PostgreSQL)

```bash
# Start PostgreSQL (using Docker)
docker run --name postgres-dev \
  -e POSTGRES_PASSWORD=devpassword \
  -e POSTGRES_DB=navisha_spark \
  -p 5432:5432 \
  -d postgres:15

# Run migrations
psql -U postgres -d navisha_spark -f migrations/001_init.sql
psql -U postgres -d navisha_spark -f migrations/002_add_key_to_insights.sql
```

### 3. Configure Environment

```bash
# Create .env for development
cp .env.example .env

# Update DATABASE_URL for local PostgreSQL
DATABASE_URL=postgresql://postgres:devpassword@localhost:5432/navisha_spark?sslmode=disable
```

### 4. Run Application

```bash
# Run directly
go run ./cmd/spark/main.go

# Or build and run
go build -o spark ./cmd/spark
./spark
```

### 5. Run Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/telegram/...

# Run with verbose output
go test -v ./...
```

---

## Production Deployment

### Option 1: Docker Compose (Recommended)

#### 1. Prepare VPS

```bash
# SSH ke VPS
ssh root@your-vps-ip

# Install Docker
curl -fsSL https://get.docker.com | sh

# Install Docker Compose
apt-get update
apt-get install -y docker-compose-plugin

# Start Docker
systemctl start docker
systemctl enable docker
```

#### 2. Deploy Application

```bash
# Clone repository
git clone https://github.com/yourusername/navisha-spark.git
cd navisha-spark

# Setup config (lihat section 3.4)
cp .env.example .env
nano .env

# Build and start
docker-compose up -d --build

# Check status
docker-compose ps

# Check logs
docker-compose logs -f spark
```

#### 3. Setup Auto-Start

```bash
# Create systemd service
sudo nano /etc/systemd/system/navisha-spark.service
```

```ini
[Unit]
Description=Navisha Spark Telegram Bot
After=docker.service
Requires=docker.service

[Service]
Type=simple
WorkingDirectory=/opt/navisha-spark
ExecStart=/usr/bin/docker-compose up
ExecStop=/usr/bin/docker-compose down
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

```bash
# Enable and start
sudo systemctl enable navisha-spark
sudo systemctl start navisha-spark

# Check status
sudo systemctl status navisha-spark

# View logs
sudo journalctl -u navisha-spark -f
```

### Option 2: Manual Binary Deployment

#### 1. Build Binary

```bash
# On your local machine
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go build -ldflags="-s -w" -o spark ./cmd/spark
```

#### 2. Transfer to VPS

```bash
# Copy binary
scp spark root@your-vps-ip:/opt/navisha-spark/

# Copy config files (hanya config.yaml, categories.yaml & schedule.yaml sudah dihapus)
scp -r config root@your-vps-ip:/opt/navisha-spark/
scp .env root@your-vps-ip:/opt/navisha-spark/
```

#### 3. Run on VPS

```bash
# SSH to VPS
ssh root@your-vps-ip

# Create systemd service
sudo nano /etc/systemd/system/navisha-spark.service
```

```ini
[Unit]
Description=Navisha Spark Telegram Bot
After=network.target

[Service]
Type=simple
User=spark
WorkingDirectory=/opt/navisha-spark
ExecStart=/opt/navisha-spark/spark
Restart=always
RestartSec=10
EnvironmentFile=/opt/navisha-spark/.env

[Install]
WantedBy=multi-user.target
```

```bash
# Enable and start
sudo systemctl enable navisha-spark
sudo systemctl start navisha-spark
```

---

## Database Setup

### Supabase Setup

#### 1. Create Supabase Project

1. Buka [Supabase](https://supabase.com/)
2. Create new project
3. Tunggu provisioning selesai (~2 menit)

#### 2. Get Connection String

1. Buka **Settings** → **Database**
2. Scroll ke **Connection string**
3. Copy **URI** (gunakan pooler port 6543 untuk production)

#### 3. Run Migrations

**Option A: Via Supabase SQL Editor (Recommended)**
```bash
# 1. Buka https://supabase.com → SQL Editor
# 2. Copy paste isi migrations/001_init.sql → Klik Run
# 3. Copy paste isi migrations/002_add_key_to_insights.sql → Klik Run
```

**Option B: Via psql**
```bash
psql $DATABASE_URL -f migrations/001_init.sql
psql $DATABASE_URL -f migrations/002_add_key_to_insights.sql
```

#### 4. Verify Tables

```sql
SELECT tablename FROM pg_tables WHERE schemaname = 'public';
-- Expected: insights, delivery_log, rotation_state, sent_history

-- Cek kolom key ada di insights
SELECT column_name FROM information_schema.columns
WHERE table_name = 'insights' AND column_name = 'key';
```

---

## Testing

### Unit Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/telegram/...

# Verbose output
go test -v ./...
```

### Integration Tests

```bash
# Run integration tests (requires database)
go test -tags=integration ./...

# Run with test database
DATABASE_URL=postgresql://localhost:5432/navisha_test?sslmode=disable go test ./...
```

---

## Troubleshooting

### 1. Container Won't Start

```bash
# Check logs
docker-compose logs spark

# Penyebab umum:
# - Port 8080 sudah dipakai → buat docker-compose.override.yml
# - DATABASE_URL salah → cek .env
# - Environment variables kurang → cek .env
```

### 2. Database Connection Failed

```bash
# Test koneksi
psql $DATABASE_URL -c "SELECT 1"

# Cek DATABASE_URL
cat .env | grep DATABASE_URL
```

### 3. Telegram Bot Not Sending Messages

```bash
# Cek bot token
curl https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getMe

# Cek chat ID
curl https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates
```

### 4. LLM Generation Fails

```bash
# Cek API key
echo $OPENROUTER_API_KEY

# Test API
curl https://openrouter.ai/api/v1/models \
  -H "Authorization: Bearer $OPENROUTER_API_KEY"
```

### 5. Health Check Failing

```bash
# Cek status container
docker inspect --format='{{.State.Health.Status}}' navisha-spark

# Test manual
curl http://localhost:8080/healthz
```

### 6. Config Changes Not Taking Effect

```bash
# Pastikan file yang diedit adalah config/config.yaml
# File categories.yaml dan schedule.yaml sudah tidak ada — semua ada di config.yaml

# Cek log untuk hot-reload
docker-compose logs spark | grep -i "config reloaded"

# Jika tidak ada log reload, cek apakah file disimpan dengan benar
cat config/config.yaml | grep -A 3 "categories:"
```

---

## Security Checklist

- [ ] `.env` file is NOT committed to Git
- [ ] `.gitignore` includes `.env`, `*.db`, `logs/`
- [ ] Telegram bot token is in `.env` only
- [ ] OpenRouter API key is in `.env` only
- [ ] Database credentials are in `.env` only
- [ ] Container runs as non-root user (`spark`)
- [ ] Firewall allows only necessary ports (22, 80, 443)
- [ ] `/trigger` endpoint dilindungi Nginx rate-limiting di production
- [ ] SSL/TLS aktif via Let's Encrypt (lihat `deploy/README.md`)

---

**Guide End**

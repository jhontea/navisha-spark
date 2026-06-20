# Docker Deployment — Skill Guide

## Overview

Skill ini berisi panduan lengkap untuk deployment Navisha Spark menggunakan Docker dan Docker Compose. Mencakup image building, container management, networking, volumes, dan best practices untuk production deployment di VPS.

---

## 1. Docker Fundamentals

### 1.1 Multi-Stage Build

Multi-stage build digunakan untuk membuat image yang kecil dan aman dengan memisahkan build environment dan runtime environment.

```dockerfile
# Stage 1: Builder
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o spark ./cmd/spark

# Stage 2: Runtime
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /app/spark .
COPY --from=builder /app/config/ ./config/
USER spark
EXPOSE 8080
CMD ["./spark"]
```

**Keuntungan:**
- Image final hanya ~15-20MB (vs ~1GB untuk golang image)
- Tidak ada build tools di production image
- Lebih aman (smaller attack surface)

### 1.2 Dockerfile Best Practices

```dockerfile
# ✅ Good practices:

# 1. Use specific base image versions
FROM golang:1.23-alpine AS builder
FROM alpine:3.19

# 2. Combine RUN commands to reduce layers
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g 1001 -S spark && \
    adduser -u 1001 -S spark -G spark

# 3. Use COPY instead of ADD (unless you need tar extraction)
COPY --from=builder /app/spark .

# 4. Use non-root user
USER spark

# 5. Use HEALTHCHECK
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/healthz || exit 1

# 6. Use CMD in exec form
CMD ["./spark"]
```

---

## 2. Docker Compose

### 2.1 Complete docker-compose.yml

```yaml
version: '3.8'

services:
  spark:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: navisha-spark
    ports:
      - "8080:8080"
    environment:
      - APP_NAME=${APP_NAME:-navisha-spark}
      - APP_ENV=${APP_ENV:-production}
      - APP_PORT=${APP_PORT:-8080}
      - LOG_LEVEL=${LOG_LEVEL:-info}
      - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
      - TELEGRAM_CHAT_ID=${TELEGRAM_CHAT_ID}
      - DATABASE_URL=${DATABASE_URL}
      - OPENROUTER_API_KEY=${OPENROUTER_API_KEY}
      - OPENROUTER_MODEL=${OPENROUTER_MODEL:-openrouter/owl-alpha}
      - SCHEDULE_CRON=${SCHEDULE_CRON:-0 */3 * * *}
      - TIMEZONE=${TIMEZONE:-Asia/Jakarta}
      - ACTIVE_HOURS_START=${ACTIVE_HOURS_START:-0}
      - ACTIVE_HOURS_END=${ACTIVE_HOURS_END:-23}
      - MAX_RETRIES=${MAX_RETRIES:-3}
      - RETRY_DELAYS=${RETRY_DELAYS:-1m,5m,15m}
      - DEDUP_WINDOW_HOURS=${DEDUP_WINDOW_HOURS:-24}
      - LEVEL_DISTRIBUTION=${LEVEL_DISTRIBUTION:-beginner:20,intermediate:50,advanced:30}
      - MIN_DAYS_BEFORE_REPEAT=${MIN_DAYS_BEFORE_REPEAT:-7}
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/healthz"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 10s
    networks:
      - spark-network
    labels:
      - "com.navisha.service=spark"
      - "com.navisha.component=telegram-bot"
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

networks:
  spark-network:
    driver: bridge

volumes:
  spark-data:
```

### 2.2 Environment Variables

```bash
# .env file (NOT committed to Git)
APP_NAME=navisha-spark
APP_ENV=production
APP_PORT=8080
LOG_LEVEL=info

TELEGRAM_BOT_TOKEN=your_telegram_bot_token_here
TELEGRAM_CHAT_ID=your_telegram_chat_id_here

DATABASE_URL=postgresql://username:password@host:6543/postgres

OPENROUTER_API_KEY=sk-or-v1-your-api-key-here
OPENROUTER_MODEL=openrouter/owl-alpha

SCHEDULE_CRON=0 */3 * * *
TIMEZONE=Asia/Jakarta
ACTIVE_HOURS_START=0
ACTIVE_HOURS_END=23

MAX_RETRIES=3
RETRY_DELAYS=1m,5m,15m
DEDUP_WINDOW_HOURS=24
LEVEL_DISTRIBUTION=beginner:20,intermediate:50,advanced:30
MIN_DAYS_BEFORE_REPEAT=7
```

---

## 3. Building & Running

### 3.1 Build Image

```bash
# Build image
docker-compose build

# Build without cache
docker-compose build --no-cache

# Build specific service
docker-compose build spark
```

### 3.2 Run Container

```bash
# Start in background
docker-compose up -d

# Start with logs
docker-compose up

# Start specific service
docker-compose up spark

# Restart service
docker-compose restart spark

# Stop service
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

### 3.3 View Logs

```bash
# Follow logs
docker-compose logs -f spark

# Last 100 lines
docker-compose logs --tail=100 spark

# With timestamps
docker-compose logs -f --timestamps spark
```

---

## 4. Container Management

### 4.1 Basic Commands

```bash
# List running containers
docker-compose ps

# List all containers
docker ps -a

# Execute command in running container
docker-compose exec spark sh

# Execute as root (if needed)
docker-compose exec -u root spark sh

# Stop container
docker-compose stop spark

# Start container
docker-compose start spark

# Remove container
docker-compose rm spark

# View container logs
docker logs navisha-spark

# Follow logs
docker logs -f navisha-spark
```

### 4.2 Health Check

```bash
# Check health status
docker inspect --format='{{.State.Health.Status}}' navisha-spark

# View health check logs
docker inspect --format='{{json .State.Health}}' navisha-spark | jq

# Manual health check
curl http://localhost:8080/healthz
```

### 4.3 Resource Usage

```bash
# View resource usage
docker stats navisha-spark

# Output:
# CONTAINER ID   NAME            CPU %     MEM USAGE / LIMIT   NET I/O
# abc123def      navisha-spark   0.05%     12.5MiB / 1GiB      1.2kB / 0B
```

---

## 5. Networking

### 5.1 Docker Networks

```yaml
# docker-compose.yml
networks:
  spark-network:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
```

### 5.2 Port Mapping

```yaml
services:
  spark:
    ports:
      - "8080:8080"  # host:container
```

**Format:** `"HOST_PORT:CONTAINER_PORT"`

**Navisha Spark:**
- Host: `8080` (health check + metrics)
- Container: `8080`

### 5.3 Network Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `bridge` | Default, isolated network | Multiple services |
| `host` | Share host network | Performance critical |
| `none` | No network | Offline processing |

**Navisha Spark menggunakan `bridge`** untuk isolation.

---

## 6. Volumes

### 6.1 Named Volumes

```yaml
volumes:
  spark-data:
    driver: local
```

### 6.2 Bind Mounts

```yaml
services:
  spark:
    volumes:
      - ./config:/app/config:ro  # Read-only
      - ./logs:/app/logs          # Read-write
```

**Navisha Spark tidak butuh volume** karena semua data disimpan di PostgreSQL (Supabase).

---

## 7. Image Optimization

### 7.1 Reduce Image Size

```dockerfile
# ✅ Good - minimal image
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata

# ❌ Bad - large image
FROM ubuntu:latest
RUN apt-get update && apt-get install -y ca-certificates tzdata
```

### 7.2 Layer Caching

```dockerfile
# ✅ Good - copy go.mod first for caching
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build

# ❌ Bad - invalidate cache on every change
COPY . .
RUN go mod download
RUN go build
```

### 7.3 Build Arguments

```dockerfile
# Dockerfile
ARG BUILD_VERSION=unknown
LABEL version=${BUILD_VERSION}

# Build with arg
docker build --build-arg BUILD_VERSION=1.0.0 .
```

---

## 8. Production Deployment

### 8.1 VPS Setup

```bash
# 1. SSH ke VPS
ssh root@your-vps-ip

# 2. Install Docker
curl -fsSL https://get.docker.com | sh

# 3. Install Docker Compose
apt-get update
apt-get install -y docker-compose-plugin

# 4. Clone repository
git clone https://github.com/yourusername/navisha-spark.git
cd navisha-spark

# 5. Create .env file
cp .env.example .env
nano .env  # Edit dengan kredensial asli

# 6. Start service
docker-compose up -d

# 7. Check logs
docker-compose logs -f spark
```

### 8.2 Systemd Service (Alternative)

```ini
# /etc/systemd/system/navisha-spark.service
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
systemctl enable navisha-spark
systemctl start navisha-spark

# Check status
systemctl status navisha-spark

# View logs
journalctl -u navisha-spark -f
```

### 8.3 Auto-Update

```bash
# Install watchtower untuk auto-update
docker run -d \
  --name watchtower \
  -v /var/run/docker.sock:/var/run/docker.sock \
  containrrr/watchtower \
  --interval 3600 \
  navisha-spark
```

---

## 9. Monitoring

### 9.1 Health Check Endpoint

```go
// internal/http/server.go
func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()

    // Check database
    if err := s.db.PingContext(ctx); err != nil {
        logrus.WithError(err).Error("Database health check failed")
        http.Error(w, "Database connection failed", http.StatusServiceUnavailable)
        return
    }

    // Check Telegram API
    if err := s.telegram.Ping(ctx); err != nil {
        logrus.WithError(err).Error("Telegram API health check failed")
        http.Error(w, "Telegram API unreachable", http.StatusServiceUnavailable)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "status":    "ok",
        "timestamp": time.Now().UTC().Format(time.RFC3339),
        "version":   s.version,
    })
}
```

### 9.2 Docker Logging

```yaml
services:
  spark:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

### 9.3 Prometheus Metrics (Optional)

```yaml
services:
  spark:
    ports:
      - "8080:8080"
    labels:
      - "prometheus.io/scrape=true"
      - "prometheus.io/port=8080"
      - "prometheus.io/path=/metrics"
```

---

## 10. Security

### 10.1 Non-Root User

```dockerfile
# Create non-root user
RUN addgroup -g 1001 -S spark && \
    adduser -u 1001 -S spark -G spark

# Switch to non-root user
USER spark
```

### 10.2 Read-Only Filesystem

```yaml
services:
  spark:
    read_only: true
    tmpfs:
      - /tmp
```

### 10.3 Capabilities

```yaml
services:
  spark:
    cap_drop:
      - ALL
    cap_add:
      - NET_BIND_SERVICE  # Only if binding to port < 1024
```

### 10.4 Secrets Management

```yaml
# docker-compose.yml
services:
  spark:
    secrets:
      - telegram_bot_token
      - openrouter_api_key

secrets:
  telegram_bot_token:
    file: ./secrets/telegram_bot_token.txt
  openrouter_api_key:
    file: ./secrets/openrouter_api_key.txt
```

```go
// Read secret from file
func ReadSecret(path string) (string, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(data)), nil
}
```

---

## 11. Backup & Restore

### 11.1 Database Backup

```bash
# Backup Supabase database
docker-compose exec spark pg_dump \
  -h aws-1-ap-southeast-1.pooler.supabase.com \
  -U postgres \
  -d postgres \
  -F c \
  -f backup.dump

# Restore
docker-compose exec spark pg_restore \
  -h aws-1-ap-southeast-1.pooler.supabase.com \
  -U postgres \
  -d postgres \
  -c backup.dump
```

### 11.2 Automated Backup Script

```bash
#!/bin/bash
# scripts/backup.sh

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/opt/navisha-spark/backups"

mkdir -p $BACKUP_DIR

# Backup database
pg_dump $DATABASE_URL -F c -f $BACKUP_DIR/backup_$DATE.dump

# Keep only last 7 days
find $BACKUP_DIR -name "backup_*.dump" -mtime +7 -delete

echo "Backup completed: backup_$DATE.dump"
```

```yaml
# Add to docker-compose.yml
services:
  backup:
    image: postgres:15
    volumes:
      - ./backups:/backups
    environment:
      - DATABASE_URL=${DATABASE_URL}
    command: >
      sh -c "pg_dump $$DATABASE_URL -F c -f /backups/backup_$$(date +%Y%m%d_%H%M%S).dump"
    networks:
      - spark-network
```

---

## 12. Troubleshooting

### 12.1 Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| Container won't start | Port already in use | Change port mapping or stop conflicting service |
| Container keeps restarting | App crash | Check logs: `docker-compose logs spark` |
| Can't connect to database | Wrong DATABASE_URL | Verify connection string in `.env` |
| Health check failing | App not ready | Increase `start_period` in healthcheck |
| Out of memory | Image too large | Optimize Dockerfile, use multi-stage build |

### 12.2 Debug Container

```bash
# Start container with override
docker-compose -f docker-compose.yml -f docker-compose.override.yml up

# docker-compose.override.yml
services:
  spark:
    environment:
      - LOG_LEVEL=debug
    command: ["sh", "-c", "./spark && sleep 3600"]
```

### 12.3 Clean Up

```bash
# Remove stopped containers
docker-compose rm

# Remove unused images
docker image prune -a

# Remove unused volumes
docker volume prune

# Full cleanup
docker system prune -a --volumes
```

---

## 13. CI/CD Integration

### 13.1 GitHub Actions

```yaml
# .github/workflows/deploy.yml
name: Deploy to VPS

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Deploy via SSH
        uses: appleboy/ssh-action@v1.0.0
        with:
          host: ${{ secrets.VPS_HOST }}
          username: ${{ secrets.VPS_USER }}
          key: ${{ secrets.VPS_SSH_KEY }}
          script: |
            cd /opt/navisha-spark
            git pull origin main
            docker-compose build
            docker-compose up -d
            docker-compose logs -f spark
```

### 13.2 Docker Registry

```bash
# Login to registry
docker login -u yourusername -p yourpassword docker.io

# Tag image
docker tag navisha-spark:latest yourusername/navisha-spark:latest

# Push image
docker push yourusername/navisha-spark:latest

# Pull and run
docker-compose pull
docker-compose up -d
```

---

## 14. Best Practices

### 14.1 Do's

✅ **Use multi-stage builds** untuk reduce image size  
✅ **Run as non-root user** untuk security  
✅ **Use health checks** untuk monitoring  
✅ **Set restart policy** untuk auto-recovery  
✅ **Use .env file** untuk secrets (don't hardcode)  
✅ **Limit resources** (CPU, memory) jika needed  
✅ **Use specific image tags** (jangan `latest` di production)  
✅ **Scan images for vulnerabilities** (`docker scan`)  
✅ **Use Docker Compose** untuk orchestration  
✅ **Log to stdout/stderr** (Docker logging driver)

### 14.2 Don'ts

❌ **Don't run as root**  
❌ **Don't hardcode secrets** di Dockerfile  
❌ **Don't use `latest` tag** di production  
❌ **Don't store data di container** (use volumes or external DB)  
❌ **Don't expose unnecessary ports**  
❌ **Don't ignore image size** (keep it minimal)  
❌ **Don't skip health checks**  
❌ **Don't use `docker-compose down -v`** di production (lose data)

---

## 15. Docker Compose Commands Cheat Sheet

```bash
# Start
docker-compose up -d

# Stop
docker-compose down

# Restart
docker-compose restart

# Logs
docker-compose logs -f

# Execute command
docker-compose exec spark sh

# Rebuild
docker-compose build

# Pull latest image
docker-compose pull

# Show running services
docker-compose ps

# Stop specific service
docker-compose stop spark

# Start specific service
docker-compose start spark

# Remove stopped containers
docker-compose rm

# View config
docker-compose config

# Validate config
docker-compose config --quiet
```

---

## 16. Performance Tuning

### 16.1 Resource Limits

```yaml
services:
  spark:
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
        reservations:
          cpus: '0.25'
          memory: 256M
```

### 16.2 Health Check Tuning

```yaml
healthcheck:
  test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/healthz"]
  interval: 30s      # Check every 30s
  timeout: 5s        # Timeout after 5s
  retries: 3         # Fail after 3 retries
  start_period: 10s  # Grace period for startup
```

---

**Document End**
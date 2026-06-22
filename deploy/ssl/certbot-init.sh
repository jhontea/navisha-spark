#!/bin/bash
# ============================================
# Certbot SSL Certificate Initial Setup
# Run this AFTER DNS is pointing to VPS
# ============================================

set -e

# Domain configuration
DOMAINS=(
    "navisha.cloud"
    "www.navisha.cloud"
    "spark.navisha.cloud"
)

# Email for Let's Encrypt notifications
EMAIL="admin@navisha.cloud"

echo "=========================================="
echo "  Certbot SSL Setup"
echo "=========================================="

# Build domain arguments
DOMAIN_ARGS=""
for domain in "${DOMAINS[@]}"; do
    DOMAIN_ARGS="$DOMAIN_ARGS -d $domain"
done

echo "[1/3] Stopping nginx container (if running)..."
docker-compose down 2>/dev/null || true

echo "[2/3] Requesting SSL certificates..."
docker-compose run --rm certbot certonly \
    --webroot \
    --webroot-path=/var/www/certbot \
    $DOMAIN_ARGS \
    --email $EMAIL \
    --agree-tos \
    --no-eff-email \
    --force-renewal

echo "[3/3] Starting all services..."
docker-compose up -d

echo ""
echo "=========================================="
echo "  SSL Setup Complete!"
echo "=========================================="
echo ""
echo "Certificates created for:"
for domain in "${DOMAINS[@]}"; do
    echo "  - $domain"
done
echo ""
echo "Test HTTPS:"
echo "  https://navisha.cloud"
echo "  https://spark.navisha.cloud/healthz"
echo ""
echo "Auto-renewal is handled by certbot container."
echo ""
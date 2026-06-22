#!/bin/bash
# ============================================
# Navisha Spark — VPS Initial Setup Script
# Run as root on fresh Ubuntu 22.04/24.04 VPS
# ============================================

set -e

echo "=========================================="
echo "  Navisha Spark — VPS Setup"
echo "=========================================="

# ============================================
# 1. System Update
# ============================================
echo "[1/8] Updating system..."
apt update && apt upgrade -y

# ============================================
# 2. Install Essential Tools
# ============================================
echo "[2/8] Installing essential tools..."
apt install -y curl wget git nano ufw software-properties-common openssh-server

# ============================================
# 3. Install Docker
# ============================================
echo "[3/8] Installing Docker..."
curl -fsSL https://get.docker.com | sh
systemctl enable docker
systemctl start docker

# ============================================
# 4. Install Docker Compose (standalone)
# ============================================
echo "[4/8] Installing Docker Compose..."
COMPOSE_VERSION=$(curl -s https://api.github.com/repos/docker/compose/releases/latest | grep tag_name | cut -d '"' -f 4)
curl -L "https://github.com/docker/compose/releases/download/${COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose

# ============================================
# 5. Install Nginx
# ============================================
echo "[5/8] Installing Nginx..."
apt install -y nginx
systemctl enable nginx

# ============================================
# 6. Install Certbot for SSL
# ============================================
echo "[6/8] Installing Certbot..."
apt install -y certbot python3-certbot-nginx

# ============================================
# 7. Setup Firewall
# ============================================
echo "[7/8] Configuring firewall..."
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable


# ============================================
# Summary
# ============================================
echo "=========================================="
echo "  Setup Complete!"
echo "=========================================="
echo ""
echo "Docker version: $(docker --version)"
echo "Docker Compose: $(docker-compose --version)"
echo "Nginx: $(nginx -v 2>&1)"
echo ""
echo "Next steps:"
echo "  1. Add SSH key to GitHub (see above)"
echo "  2. Clone repo: git clone git@github.com:jhontea/navisha-spark.git"
echo "  3. Setup .env file"
echo "  4. Run docker-compose up -d"
echo "  5. Setup Nginx reverse proxy"
echo "  6. Setup SSL with Certbot"
echo ""
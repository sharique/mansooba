#!/bin/bash
# EC2 bootstrap script — runs once on first boot as root.
# Rendered by Terraform's templatefile() with aws_region, smtp_host,
# smtp_port, and smtp_from substituted before the script reaches the instance.
#
# What this script does:
#   1. Installs Docker and the AWS CLI
#   2. Fetches all secrets from SSM Parameter Store (no secrets in this file)
#   3. Writes /opt/mansooba/.env with all runtime configuration
#   4. Logs in to GHCR using the stored PAT
#   5. Pulls and starts the compose.prod.yml stack
#
# Debug: sudo tail -f /var/log/user-data.log

set -euo pipefail
exec > >(tee /var/log/user-data.log | logger -t user-data -s 2>/dev/console) 2>&1

echo "=== Mansooba EC2 Bootstrap ==="
echo "Region: ${aws_region}"

# ── Install Docker ────────────────────────────────────────────────────────────
# Amazon Linux 2023 uses dnf. The AWS CLI v2 and curl are pre-installed.
# docker-compose-plugin is not in the AL2023 repos — install the binary directly
# into Docker's CLI plugin directory so `docker compose` (v2) works.
dnf update -y
dnf install -y docker

systemctl enable docker
systemctl start docker

# Install Docker Compose v2 plugin
COMPOSE_DIR=/usr/local/lib/docker/cli-plugins
mkdir -p $COMPOSE_DIR
curl -fsSL "https://github.com/docker/compose/releases/latest/download/docker-compose-linux-x86_64" \
  -o "$COMPOSE_DIR/docker-compose"
chmod +x "$COMPOSE_DIR/docker-compose"

# Allow ec2-user to run docker without sudo (takes effect on next login).
usermod -aG docker ec2-user

# ── Create app directory ──────────────────────────────────────────────────────
mkdir -p /opt/mansooba
cd /opt/mansooba

# ── Fetch secrets from SSM Parameter Store ───────────────────────────────────
# The EC2 instance role grants ssm:GetParameter on /mansooba/* (see iam module).
# Using --with-decryption to read SecureString values.
echo "Fetching secrets from SSM..."

get_param() {
  aws ssm get-parameter \
    --name "$1" \
    --with-decryption \
    --region "${aws_region}" \
    --query Parameter.Value \
    --output text
}

JWT_SECRET=$(get_param /mansooba/JWT_SECRET)
DB_PASSWORD=$(get_param /mansooba/DB_PASSWORD)
GHCR_PAT=$(get_param /mansooba/GHCR_PAT)
RDS_ENDPOINT=$(get_param /mansooba/RDS_ENDPOINT)

# SES SMTP credentials — created by the Terraform SES module and stored in SSM.
# SMTP_USER is the IAM access key ID; SMTP_PASS is the derived SMTP password
# (ses_smtp_password_v4), NOT the raw IAM secret key.
SMTP_USER=$(get_param /mansooba/SMTP_USER)
SMTP_PASS=$(get_param /mansooba/SMTP_PASS)

# ── Resolve public IP for CORS and magic-link base URL ───────────────────────
# The instance metadata service (169.254.169.254) provides the public IPv4.
# This is the same IP the Elastic IP will point to; using it directly avoids
# a Terraform circular dependency between the EIP and an SSM parameter.
PUBLIC_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4)
APP_BASE_URL="http://$${PUBLIC_IP}"

# ── Write .env ────────────────────────────────────────────────────────────────
# chmod 600 prevents other OS users from reading secrets.
# The file is bind-mounted into the backend container via env_file in compose.prod.yml.
cat > /opt/mansooba/.env <<EOF
# ── Server ────────────────────────────────────────────────────────────────────
SERVER_PORT=8080
APP_ENV=production
LOG_LEVEL=info

# ── Auth ─────────────────────────────────────────────────────────────────────
JWT_SECRET=$${JWT_SECRET}
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h
REVOKED_TOKEN_CLEANUP_INTERVAL=15m

# ── CORS ─────────────────────────────────────────────────────────────────────
# Must match the origin the browser uses. Update if you add a custom domain.
CORS_ORIGINS=$${APP_BASE_URL}

# ── Database ──────────────────────────────────────────────────────────────────
# sslmode=require enforces TLS in transit to RDS.
DB_DRIVER=postgres
DB_DSN=host=$${RDS_ENDPOINT} user=mansooba password=$${DB_PASSWORD} dbname=mansooba port=5432 sslmode=require
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m

# ── Email (AWS SES via SMTP) ──────────────────────────────────────────────────
# SMTP_HOST is injected by Terraform templatefile() as the SES regional endpoint.
# SMTP_USER / SMTP_PASS are fetched from SSM (derived SES SMTP credentials).
# Leaving SMTP_HOST empty would activate NoopSender — all emails silently dropped.
SMTP_HOST=${smtp_host}
SMTP_PORT=${smtp_port}
SMTP_FROM=${smtp_from}
SMTP_USER=$${SMTP_USER}
SMTP_PASS=$${SMTP_PASS}

# ── Magic links ───────────────────────────────────────────────────────────────
# Used to construct clickable password-reset URLs in emails.
# Update to your custom domain if you add one later.
APP_BASE_URL=$${APP_BASE_URL}
EOF

chmod 600 /opt/mansooba/.env

# ── Log in to GHCR ────────────────────────────────────────────────────────────
# The PAT needs only the read:packages scope.
# If your GHCR packages are public, this step is optional but harmless.
echo "Logging in to GHCR..."
echo "$${GHCR_PAT}" | docker login ghcr.io -u github-actions --password-stdin

# ── Pull compose.prod.yml from GitHub ────────────────────────────────────────
# Fetches the production compose file from the main branch of the code repo.
# This file defines the backend and frontend services using GHCR images.
curl -fsSL \
  "https://raw.githubusercontent.com/sharique/mansooba/main/compose.prod.yml" \
  -o /opt/mansooba/compose.prod.yml

# ── Start the application ─────────────────────────────────────────────────────
echo "Pulling images and starting Mansooba stack..."
docker compose -f /opt/mansooba/compose.prod.yml pull
docker compose -f /opt/mansooba/compose.prod.yml up -d

echo "=== Bootstrap complete ==="
echo "App running at $${APP_BASE_URL}"
echo "Health check: curl $${APP_BASE_URL}/api/v1/health"

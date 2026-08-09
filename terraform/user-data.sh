#!/bin/bash
# EC2 bootstrap script — runs once on first boot as root.
# Rendered by Terraform's templatefile() with aws_region, smtp_host,
# smtp_port, smtp_from, storage_bucket, and rds_identifier substituted
# before the script reaches the instance.
#
# What this script does:
#   1. Installs Docker and the AWS CLI
#   2. Fetches all secrets from SSM Parameter Store (no secrets in this file)
#   3. Writes /opt/mansooba/.env with all runtime configuration
#   4. Pulls and starts the compose.prod.yml stack (GHCR images are public —
#      no docker login needed)
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

# ── System Logs (011-system-logs, ADR-031) ────────────────────────────────────
# Loki's -runtime-config.file must point at a file that exists before Loki
# starts (an empty "overrides: {}" is a valid, inert starting point — the
# backend's retention-sync reconciler rewrites it after boot). Bind-mounted
# into both the backend and loki containers via compose.prod.yml.
mkdir -p /opt/mansooba/loki
cat > /opt/mansooba/loki/runtime-overrides.yaml <<'EOF'
overrides: {}
EOF

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
RDS_ENDPOINT=$(get_param /mansooba/RDS_ENDPOINT)

# SES SMTP credentials — created by the Terraform SES module and stored in SSM.
# SMTP_USER is the IAM access key ID; SMTP_PASS is the derived SMTP password
# (ses_smtp_password_v4), NOT the raw IAM secret key.
SMTP_USER=$(get_param /mansooba/SMTP_USER)
SMTP_PASS=$(get_param /mansooba/SMTP_PASS)

# Grafana admin password (011-system-logs) — optional feature (the
# "observability" Compose profile), so this must NOT hard-fail bootstrap
# when the operator hasn't created the SSM parameter. Falls back to a
# freshly generated password every boot; set /mansooba/GRAFANA_ADMIN_PASSWORD
# in SSM for a password that's stable across instance restarts.
GRAFANA_ADMIN_PASSWORD=$(get_param /mansooba/GRAFANA_ADMIN_PASSWORD 2>/dev/null || head -c 24 /dev/urandom | base64)

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

# ── Database idle auto-stop / wake-on-hit (feature 010) ───────────────────────
# RDS_INSTANCE_IDENTIFIER must match the leading label of RDS_ENDPOINT's hostname
# for Config.RDSAutoStopApplies() to engage — see ADR-030 and rds_hostname.go.
RDS_INSTANCE_IDENTIFIER=${rds_identifier}
# AWS_REGION is required for the RDS SDK client (rdsclient.go) — aws-sdk-go-v2's
# LoadDefaultConfig does NOT fall back to EC2 instance metadata for region (only
# for credentials); without this the backend fails fast at startup with a
# "missing region" error whenever auto-stop is enabled.
AWS_REGION=${aws_region}

# ── System Logs (011-system-logs, ADR-031) ────────────────────────────────────
# LOKI_BASE_URL uses the compose service name, same pattern as local dev.
# LOKI_RUNTIME_OVERRIDES_PATH must match the container-side path in
# compose.prod.yml's volume mounts for both the backend and loki services.
LOKI_BASE_URL=http://loki:3100
LOKI_RUNTIME_OVERRIDES_PATH=/etc/loki/runtime-overrides.yaml
LOKI_RETENTION_SYNC_INTERVAL=5m

# ── Grafana (optional, 011-system-logs) ───────────────────────────────────────
# Only takes effect if the "observability" Compose profile is started
# manually (see compose.prod.yml's comment) — the backend never reads
# these. GF_* are Grafana's own native env var names, read directly by the
# grafana container via compose.prod.yml's env_file.
GF_SECURITY_ADMIN_USER=admin
GF_SECURITY_ADMIN_PASSWORD=$${GRAFANA_ADMIN_PASSWORD}
GF_AUTH_ANONYMOUS_ENABLED=false
GF_SERVER_ROOT_URL=http://$${PUBLIC_IP}:3001/

# ── Attachment storage (S3) ────────────────────────────────────────────────────
# No access key/secret here: the SDK authenticates via the EC2 instance's IAM
# role (see modules/iam). Leaving STORAGE_ENDPOINT unset means "real AWS S3".
STORAGE_BUCKET=${storage_bucket}
STORAGE_REGION=${aws_region}

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

# ── Pull compose.prod.yml from GitHub ────────────────────────────────────────
# Fetches the production compose file from the main branch of the code repo.
# This file defines the backend and frontend services using GHCR images.
curl -fsSL \
  "https://raw.githubusercontent.com/sharique/mansooba/main/compose.prod.yml" \
  -o /opt/mansooba/compose.prod.yml

# Loki's static config (011-system-logs) — same file compose.yml uses
# locally; see its own comment for why max_query_length is overridden.
# runtime-overrides.yaml was already written above, by this script itself
# (not fetched — the backend's retention-sync reconciler owns that file).
curl -fsSL \
  "https://raw.githubusercontent.com/sharique/mansooba/main/loki/local-config.yaml" \
  -o /opt/mansooba/loki/local-config.yaml

# Grafana's datasource + dashboard provisioning (optional, 011-system-logs)
# — only read if the "observability" profile is started; harmless to fetch
# unconditionally, same reasoning as the Loki config above.
mkdir -p /opt/mansooba/grafana/provisioning/datasources /opt/mansooba/grafana/provisioning/dashboards
for f in grafana/provisioning/datasources/loki.yaml \
         grafana/provisioning/dashboards/dashboards.yaml \
         grafana/provisioning/dashboards/system-logs.json \
         grafana/provisioning/dashboards/container-logs.json; do
  curl -fsSL "https://raw.githubusercontent.com/sharique/mansooba/main/$f" \
    -o "/opt/mansooba/$f"
done

# Alloy's Docker-log-collection config (011-system-logs) — unlike Grafana,
# this one IS started by default, so it must exist before `compose up` runs.
mkdir -p /opt/mansooba/alloy
curl -fsSL \
  "https://raw.githubusercontent.com/sharique/mansooba/main/alloy/config.alloy" \
  -o /opt/mansooba/alloy/config.alloy

# ── Start the application ─────────────────────────────────────────────────────
echo "Pulling images and starting Mansooba stack..."
docker compose -f /opt/mansooba/compose.prod.yml pull
docker compose -f /opt/mansooba/compose.prod.yml up -d

echo "=== Bootstrap complete ==="
echo "App running at $${APP_BASE_URL}"
echo "Health check: curl $${APP_BASE_URL}/api/v1/health"

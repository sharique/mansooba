#!/bin/bash
set -euo pipefail

# Log to /var/log/user-data.log for debugging.
exec > >(tee /var/log/user-data.log | logger -t user-data -s 2>/dev/console) 2>&1

echo "=== Mansooba EC2 Bootstrap ==="
echo "Region: ${aws_region}"

# ── Install Docker ────────────────────────────────────────────────────────────
apt-get update -y
apt-get install -y \
  docker.io \
  docker-compose-plugin \
  awscli \
  curl \
  wget

systemctl enable docker
systemctl start docker

# Allow ubuntu user to run docker without sudo.
usermod -aG docker ubuntu

# ── Create app directory ──────────────────────────────────────────────────────
mkdir -p /opt/mansooba
cd /opt/mansooba

# ── Fetch secrets from SSM Parameter Store ───────────────────────────────────
echo "Fetching secrets from SSM..."

JWT_SECRET=$(aws ssm get-parameter \
  --name /mansooba/JWT_SECRET \
  --with-decryption \
  --region "${aws_region}" \
  --query Parameter.Value \
  --output text)

DB_PASSWORD=$(aws ssm get-parameter \
  --name /mansooba/DB_PASSWORD \
  --with-decryption \
  --region "${aws_region}" \
  --query Parameter.Value \
  --output text)

GHCR_PAT=$(aws ssm get-parameter \
  --name /mansooba/GHCR_PAT \
  --with-decryption \
  --region "${aws_region}" \
  --query Parameter.Value \
  --output text)

RDS_ENDPOINT=$(aws ssm get-parameter \
  --name /mansooba/RDS_ENDPOINT \
  --region "${aws_region}" \
  --query Parameter.Value \
  --output text)

# ── Write .env ────────────────────────────────────────────────────────────────
PUBLIC_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4)

cat > /opt/mansooba/.env <<EOF
DB_DRIVER=postgres
DB_DSN=host=$${RDS_ENDPOINT} user=mansooba password=$${DB_PASSWORD} dbname=mansooba port=5432 sslmode=require
JWT_SECRET=$${JWT_SECRET}
LOG_LEVEL=info
CORS_ORIGINS=http://$${PUBLIC_IP}
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m
EOF

chmod 600 /opt/mansooba/.env

# ── Authenticate to GHCR ──────────────────────────────────────────────────────
echo "$${GHCR_PAT}" | docker login ghcr.io -u github-actions --password-stdin

# ── Pull compose.prod.yml from GitHub ────────────────────────────────────────
curl -fsSL \
  "https://raw.githubusercontent.com/sharique/mansooba/main/compose.prod.yml" \
  -o /opt/mansooba/compose.prod.yml

# ── Start the application ─────────────────────────────────────────────────────
echo "Starting Mansooba stack..."
docker compose -f /opt/mansooba/compose.prod.yml pull
docker compose -f /opt/mansooba/compose.prod.yml up -d

echo "=== Bootstrap complete ==="
echo "App running at http://$${PUBLIC_IP}"

# ── EC2 Security Group ────────────────────────────────────────────────────────
# Controls inbound traffic to the EC2 instance.
# Egress is fully open so the instance can reach GHCR, SSM, and SES SMTP
# (all outbound traffic exits through the internet gateway).

resource "aws_security_group" "ec2" {
  name        = "${var.name_prefix}-ec2"
  description = "Allow HTTP and SSH inbound; all outbound."
  vpc_id      = var.vpc_id

  # SSH — restrict allowed_ssh_cidr to your IP (e.g. 1.2.3.4/32) for better
  # security. Default 0.0.0.0/0 allows any IP to attempt SSH.
  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.allowed_ssh_cidr]
  }

  # HTTP — served by the frontend nginx container on port 80.
  ingress {
    description = "HTTP"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Backend API — exposed directly for debugging (curl /api/v1/health).
  # Remove this rule in a hardened setup where all traffic goes through port 80.
  ingress {
    description = "Backend API (direct, for debugging)"
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Allow all outbound — required for Docker pulls (GHCR/443), SSM API calls,
  # and SES SMTP (port 587). Narrow this down only if you add a NAT gateway
  # with explicit egress rules.
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "${var.name_prefix}-ec2-sg" }
}

# ── RDS Security Group ────────────────────────────────────────────────────────
# PostgreSQL is only reachable from EC2 — never from the internet.
# Source is the EC2 security group (not a CIDR), so only instances in that
# group can connect regardless of which IPs they have.

resource "aws_security_group" "rds" {
  name        = "${var.name_prefix}-rds"
  description = "Allow PostgreSQL from EC2 security group only."
  vpc_id      = var.vpc_id

  ingress {
    description     = "PostgreSQL from EC2"
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.ec2.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "${var.name_prefix}-rds-sg" }
}

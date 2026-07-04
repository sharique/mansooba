# ── RDS Subnet Group ──────────────────────────────────────────────────────────
# AWS requires a subnet group (at least 2 AZs) for RDS even when running a
# single-AZ instance. This gives RDS the option to failover to the second AZ
# during maintenance or an AZ outage.

resource "aws_db_subnet_group" "main" {
  name       = "${var.name_prefix}-db"
  subnet_ids = var.private_subnet_ids
  tags       = { Name = "${var.name_prefix}-db-subnet-group" }
}

# ── RDS PostgreSQL Instance ───────────────────────────────────────────────────
# db.t3.micro with 20 GB gp2 storage is within the free tier
# (750 hours/month and 20 GB for 12 months on a new account).
#
# Security:
#   publicly_accessible = false — no internet-facing endpoint; only reachable
#     from within the VPC via the RDS security group (EC2 → port 5432).
#   sslmode=require in the DB_DSN enforces TLS in transit.
#
# Lifecycle:
#   skip_final_snapshot = true — no snapshot on deletion (saves storage cost
#     for dev/staging; set to false and provide final_snapshot_identifier
#     for production databases you care about recovering).
#   deletion_protection = false — allows `terraform destroy` to clean up.
#     Set to true on production to prevent accidental deletion.

resource "aws_db_instance" "postgres" {
  identifier             = "${var.name_prefix}-db"
  engine                 = "postgres"
  engine_version         = "16"
  instance_class         = var.instance_class
  allocated_storage      = var.allocated_storage
  storage_type           = "gp2"
  db_name                = var.db_name
  username               = var.db_username
  password               = var.db_password
  db_subnet_group_name   = aws_db_subnet_group.main.name
  vpc_security_group_ids = [var.security_group_id]
  publicly_accessible    = false
  skip_final_snapshot    = true
  deletion_protection    = false

  tags = { Name = "${var.name_prefix}-db" }
}

# ── Terraform Configuration ───────────────────────────────────────────────────
# Requires Terraform >= 1.7 for the templatefile() built-in used in the
# compute module's user_data argument.
# State is stored locally (terraform.tfstate). For team use, migrate to an
# S3 backend with DynamoDB locking — see terraform/README.md.

terraform {
  required_version = ">= 1.7"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

# ── Networking ────────────────────────────────────────────────────────────────
# Creates a dedicated VPC (10.0.0.0/16) with:
#   • one public subnet  (10.0.1.0/24) for EC2
#   • two private subnets (10.0.2-3.0/24) for RDS (must span 2 AZs)
#   • internet gateway + public route table so EC2 can reach the internet

module "networking" {
  source     = "./modules/networking"
  aws_region = var.aws_region
}

# ── Security Groups ───────────────────────────────────────────────────────────
# EC2 security group:  allows SSH (port 22) from allowed_ssh_cidr,
#                      HTTP (80) and backend API (8080) from anywhere.
# RDS security group:  allows PostgreSQL (5432) from the EC2 SG only —
#                      the database is never reachable from the internet.

module "security" {
  source           = "./modules/security"
  vpc_id           = module.networking.vpc_id
  allowed_ssh_cidr = var.allowed_ssh_cidr
}

# ── Attachment Storage (S3) ────────────────────────────────────────────────────
# Creates the S3 bucket that issue attachments are stored in — private,
# encrypted at rest (SSE-S3), no versioning. The backend authenticates against
# it via the EC2 instance's IAM role (see module "iam" below), never a static
# access key.

module "storage" {
  source      = "./modules/storage"
  aws_region  = var.aws_region
  name_prefix = var.s3_name_prefix
}

# ── IAM ───────────────────────────────────────────────────────────────────────
# Creates an EC2 instance role + instance profile with two inline policies:
#   • ssm:GetParameter / ssm:GetParametersByPath on /mansooba/* (read-only)
#   • s3:PutObject / GetObject / DeleteObject / DeleteObjects on the
#     attachments bucket only
# This is how the EC2 boot script fetches secrets, and how the backend
# accesses S3, without hardcoding any credentials.

module "iam" {
  source                 = "./modules/iam"
  aws_region             = var.aws_region
  ssm_path_prefix        = "/mansooba"
  attachments_bucket_arn = module.storage.bucket_arn
  db_instance_arn        = module.database.rds_arn
}

# ── Email (SES) ───────────────────────────────────────────────────────────────
# Creates (only when var.enable_ses is true — default):
#   • an SES email identity for the sender address (verification email is sent
#     to smtp_from — you must click the link before SES can send from it)
#   • an IAM user with ses:SendRawEmail permission (required for SMTP auth)
#   • an IAM access key; derives the SMTP password via ses_smtp_password_v4
#   • SSM SecureString params at /mansooba/SMTP_USER and /mansooba/SMTP_PASS
#
# After apply: check your inbox at var.smtp_from and click "Verify this email".
# To send to arbitrary addresses (not just verified ones), request production
# access in the SES Console → Account dashboard → Request production access.
#
# Set enable_ses = false to skip all of this entirely — the backend already
# handles no SMTP config gracefully (NoopSender: password-reset tokens are
# returned directly in the API response instead of emailed), so this is a
# legitimate way to run without email, not just a local-dev shortcut.

module "ses" {
  count           = var.enable_ses ? 1 : 0
  source          = "./modules/ses"
  aws_region      = var.aws_region
  smtp_from       = var.smtp_from
  ssm_path_prefix = "/mansooba"
}

# ── Compute ───────────────────────────────────────────────────────────────────
# Launches a t2.micro EC2 instance (free-tier) with:
#   • latest Ubuntu 24.04 LTS AMI (auto-resolved by the module)
#   • an auto-assigned public IP (no Elastic IP — see modules/compute/main.tf
#     for why, and terraform/README.md for how to add one back if your
#     account allows it)
#   • user-data bootstrap script that installs Docker, fetches secrets from SSM,
#     writes .env, and starts the compose.prod.yml stack (GHCR images are
#     public — no docker login needed)
#
# The user_data argument is rendered here (not inside the module) so that
# all templatefile() variables are in one place.

module "compute" {
  source                = "./modules/compute"
  aws_region            = var.aws_region
  subnet_id             = module.networking.public_subnet_id
  security_group_id     = module.security.ec2_sg_id
  instance_profile_name = module.iam.instance_profile_name
  ssh_public_key        = var.ssh_public_key
  user_data = templatefile("${path.root}/user-data.sh", {
    aws_region              = var.aws_region
    smtp_host               = var.enable_ses ? module.ses[0].smtp_host : ""
    smtp_port               = var.enable_ses ? module.ses[0].smtp_port : ""
    smtp_from               = var.enable_ses ? module.ses[0].smtp_from : ""
    storage_bucket          = module.storage.bucket_name
    rds_identifier          = module.database.rds_identifier
    rds_autostop_enabled    = var.rds_autostop_enabled
    rds_idle_timeout        = var.rds_idle_timeout
    rds_idle_check_interval = var.rds_idle_check_interval
    rds_start_failure_bound = var.rds_start_failure_bound
  })

  # user-data.sh fetches JWT_SECRET/DB_PASSWORD/RDS_ENDPOINT from SSM at boot,
  # but that's an AWS CLI call inside a shell script — invisible to Terraform's
  # own dependency graph. Without this, apply could launch the instance before
  # the SSM parameters below exist, and first-boot bootstrap would fail with
  # ParameterNotFound (as it did — the parameters simply didn't exist yet).
  depends_on = [
    aws_ssm_parameter.jwt_secret,
    aws_ssm_parameter.db_password,
    aws_ssm_parameter.rds_endpoint,
  ]
}

# ── Database ──────────────────────────────────────────────────────────────────
# Creates a db.t3.micro RDS PostgreSQL 16 instance (free-tier) inside the
# private subnets. Never publicly accessible — only reachable from EC2 via
# the RDS security group.

module "database" {
  source             = "./modules/database"
  private_subnet_ids = module.networking.private_subnet_ids
  security_group_id  = module.security.rds_sg_id
  db_password        = var.db_password
}

# ── Secrets in SSM Parameter Store ────────────────────────────────────────────
# Writes everything user-data.sh needs at boot so a single `terraform apply`
# fully bootstraps the instance — no manual `aws ssm put-parameter` step
# before or after apply.
#
# Trade-off: these values land in terraform.tfstate in plaintext — marking an
# output/variable `sensitive` only redacts CLI output, not the state file
# itself. This isn't a new exposure though: db_password already ends up in
# state via aws_db_instance's own `password` argument (modules/database)
# regardless of what we do here. Use an encrypted remote backend for anything
# beyond solo/local use — see README.md's "Upgrading to S3 backend".

resource "random_password" "jwt_secret" {
  length  = 32
  special = false
}

resource "aws_ssm_parameter" "jwt_secret" {
  name        = "/mansooba/JWT_SECRET"
  description = "JWT signing secret, generated by Terraform (random_password.jwt_secret)."
  type        = "SecureString"
  value       = random_password.jwt_secret.result
}

resource "aws_ssm_parameter" "db_password" {
  name        = "/mansooba/DB_PASSWORD"
  description = "RDS PostgreSQL master password — same value as var.db_password (module.database's aws_db_instance reads that directly; this copy exists only for user-data.sh to fetch at boot)."
  type        = "SecureString"
  value       = var.db_password
}

resource "aws_ssm_parameter" "rds_endpoint" {
  name        = "/mansooba/RDS_ENDPOINT"
  description = "RDS PostgreSQL hostname, written automatically once module.database creates the instance."
  type        = "String"
  value       = module.database.rds_endpoint
}

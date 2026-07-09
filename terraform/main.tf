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
  source     = "./modules/storage"
  aws_region = var.aws_region
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
}

# ── Email (SES) ───────────────────────────────────────────────────────────────
# Creates:
#   • an SES email identity for the sender address (verification email is sent
#     to smtp_from — you must click the link before SES can send from it)
#   • an IAM user with ses:SendRawEmail permission (required for SMTP auth)
#   • an IAM access key; derives the SMTP password via ses_smtp_password_v4
#   • SSM SecureString params at /mansooba/SMTP_USER and /mansooba/SMTP_PASS
#
# After apply: check your inbox at var.smtp_from and click "Verify this email".
# To send to arbitrary addresses (not just verified ones), request production
# access in the SES Console → Account dashboard → Request production access.

module "ses" {
  source          = "./modules/ses"
  aws_region      = var.aws_region
  smtp_from       = var.smtp_from
  ssm_path_prefix = "/mansooba"
}

# ── Compute ───────────────────────────────────────────────────────────────────
# Launches a t2.micro EC2 instance (free-tier) with:
#   • latest Ubuntu 24.04 LTS AMI (auto-resolved by the module)
#   • an Elastic IP so the public address survives restarts
#   • user-data bootstrap script that installs Docker, fetches secrets from SSM,
#     writes .env, logs in to GHCR, and starts the compose.prod.yml stack
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
    aws_region = var.aws_region
    smtp_host  = module.ses.smtp_host
    smtp_port  = module.ses.smtp_port
    smtp_from  = module.ses.smtp_from
  })
}

# ── Database ──────────────────────────────────────────────────────────────────
# Creates a db.t3.micro RDS PostgreSQL 16 instance (free-tier) inside the
# private subnets. Never publicly accessible — only reachable from EC2 via
# the RDS security group.
#
# After apply: store the RDS endpoint in SSM manually (see outputs):
#   terraform output -raw rds_endpoint | xargs -I{} \
#     aws ssm put-parameter --name /mansooba/RDS_ENDPOINT --value {} \
#     --type String --region <region>

module "database" {
  source             = "./modules/database"
  private_subnet_ids = module.networking.private_subnet_ids
  security_group_id  = module.security.rds_sg_id
  db_password        = var.db_password
}

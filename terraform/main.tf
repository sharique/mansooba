terraform {
  required_version = ">= 1.7"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  # Using local state for simplicity.
  # To migrate to S3 backend (recommended for teams), see terraform/README.md.
}

provider "aws" {
  region = var.aws_region
}

# ── Networking ────────────────────────────────────────────────────────────────

module "networking" {
  source     = "./modules/networking"
  aws_region = var.aws_region
}

# ── Security Groups ───────────────────────────────────────────────────────────

module "security" {
  source           = "./modules/security"
  vpc_id           = module.networking.vpc_id
  allowed_ssh_cidr = var.allowed_ssh_cidr
}

# ── IAM ───────────────────────────────────────────────────────────────────────

module "iam" {
  source          = "./modules/iam"
  aws_region      = var.aws_region
  ssm_path_prefix = "/mansooba"
}

# ── Compute ───────────────────────────────────────────────────────────────────

module "compute" {
  source                = "./modules/compute"
  aws_region            = var.aws_region
  subnet_id             = module.networking.public_subnet_id
  security_group_id     = module.security.ec2_sg_id
  instance_profile_name = module.iam.instance_profile_name
  ssh_public_key        = var.ssh_public_key
  user_data = templatefile("${path.root}/user-data.sh", {
    aws_region = var.aws_region
  })
}

# ── Database ──────────────────────────────────────────────────────────────────

module "database" {
  source             = "./modules/database"
  private_subnet_ids = module.networking.private_subnet_ids
  security_group_id  = module.security.rds_sg_id
  db_password        = var.db_password
}

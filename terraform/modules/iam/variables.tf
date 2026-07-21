variable "aws_region" {
  description = "AWS region — used to scope the SSM IAM policy ARN."
  type        = string
}

variable "ssm_path_prefix" {
  description = "SSM Parameter Store path prefix the EC2 instance is allowed to read (e.g. \"/mansooba\")."
  type        = string
  default     = "/mansooba"
}

variable "name_prefix" {
  description = "Prefix applied to all IAM resource names."
  type        = string
  default     = "mansooba"
}

variable "attachments_bucket_arn" {
  description = "ARN of the S3 attachments bucket (module.storage.bucket_arn) — scopes the EC2 role's S3 policy to this bucket only."
  type        = string
}

variable "db_instance_arn" {
  description = "ARN of the RDS PostgreSQL instance (module.database.rds_arn) — scopes the EC2 role's RDS lifecycle policy to this instance only (feature 010, db-idle-autostop)."
  type        = string
}

variable "aws_region" {
  description = "AWS region to deploy all resources into. Must match the region used in `aws configure`."
  type        = string
  default     = "us-east-1"
}

variable "ssh_public_key" {
  description = "SSH public key to install on the EC2 instance. Paste the full contents of ~/.ssh/mansooba.pub (starts with 'ssh-ed25519 ...')."
  type        = string
}

variable "db_password" {
  description = "Master password for the RDS PostgreSQL instance. Must also be stored in SSM as /mansooba/DB_PASSWORD — these two values must match."
  type        = string
  sensitive   = true
}

variable "enable_ses" {
  description = "Whether to provision SES (email identity, SMTP IAM user, SSM SMTP credentials). When false, no SES resources are created and smtp_from is unused — the backend falls back to NoopSender (password-reset tokens are returned directly in the API response instead of emailed), matching what happens locally when SMTP_HOST is unset."
  type        = bool
  default     = true
}

variable "smtp_from" {
  description = "Sender email address for password-reset emails (e.g. noreply@yourdomain.com). Required only when enable_ses is true — Terraform registers this with SES and AWS sends a verification email, which you must click before SES can send from it. Ignored when enable_ses is false."
  type        = string
  default     = ""
}

variable "allowed_ssh_cidr" {
  description = "CIDR block allowed to SSH to the EC2 instance. Default 0.0.0.0/0 allows all IPs. Restrict to your IP (e.g. 1.2.3.4/32) for better security: run `curl ifconfig.me` to find it."
  type        = string
  default     = "0.0.0.0/0"
}

# ── Database idle auto-stop / wake-on-hit (feature 010, ADR-030) ──────────────
# Previously only configurable via the backend's own hardcoded defaults —
# these were invisible from Terraform and undocumented for anyone deploying
# from this code. Defaults here match the backend's own (pkg/config/config.go)
# so leaving them unset changes nothing about current behavior.

variable "rds_autostop_enabled" {
  description = "Enable RDS idle auto-stop/wake-on-hit. Only takes effect when DB_DSN's hostname is confirmed as the specific RDS instance named by rds_identifier (module.database.rds_identifier) — always a no-op otherwise. Set false to disable the feature entirely on this deployment."
  type        = bool
  default     = true
}

variable "rds_idle_timeout" {
  description = "How long the database can sit idle before RDSAutoStop stops it (Go duration string, e.g. \"10m\")."
  type        = string
  default     = "10m"
}

variable "rds_idle_check_interval" {
  description = "How often the backend's background goroutine checks for idle/pending-start (Go duration string, e.g. \"1m\")."
  type        = string
  default     = "1m"
}

variable "rds_start_failure_bound" {
  description = "Consecutive failed RDS start attempts before the wake-on-hit middleware gives up and returns a plain error instead of a \"waking up\" response."
  type        = number
  default     = 3
}

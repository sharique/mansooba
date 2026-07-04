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

variable "smtp_from" {
  description = "Sender email address for password-reset emails (e.g. noreply@yourdomain.com). Terraform registers this with SES and AWS sends a verification email — you must click the link before SES can send from it."
  type        = string
}

variable "allowed_ssh_cidr" {
  description = "CIDR block allowed to SSH to the EC2 instance. Default 0.0.0.0/0 allows all IPs. Restrict to your IP (e.g. 1.2.3.4/32) for better security: run `curl ifconfig.me` to find it."
  type        = string
  default     = "0.0.0.0/0"
}

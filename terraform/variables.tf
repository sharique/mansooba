variable "aws_region" {
  description = "AWS region to deploy into."
  type        = string
  default     = "us-east-1"
}

variable "ssh_public_key" {
  description = "SSH public key to install on the EC2 instance (contents of ~/.ssh/mansooba.pub)."
  type        = string
}

variable "db_password" {
  description = "Master password for the RDS PostgreSQL instance."
  type        = string
  sensitive   = true
}

variable "allowed_ssh_cidr" {
  description = "CIDR block allowed to SSH to the EC2 instance. Default allows all IPs (0.0.0.0/0). Restrict to your IP for better security."
  type        = string
  default     = "0.0.0.0/0"
}

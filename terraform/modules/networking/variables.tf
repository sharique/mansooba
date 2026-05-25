variable "aws_region" {
  description = "AWS region to deploy into (used for availability zone suffixes)."
  type        = string
}

variable "name_prefix" {
  description = "Prefix applied to all resource Name tags. Change to deploy multiple stacks in one account."
  type        = string
  default     = "mansooba"
}

variable "vpc_cidr" {
  description = "CIDR block for the VPC."
  type        = string
  default     = "10.0.0.0/16"
}

variable "public_subnet_cidr" {
  description = "CIDR block for the public subnet (EC2 + Elastic IP)."
  type        = string
  default     = "10.0.1.0/24"
}

variable "private_subnet_a_cidr" {
  description = "CIDR block for private subnet A (region-a). Required by RDS subnet group."
  type        = string
  default     = "10.0.2.0/24"
}

variable "private_subnet_b_cidr" {
  description = "CIDR block for private subnet B (region-b). Required by RDS subnet group."
  type        = string
  default     = "10.0.3.0/24"
}

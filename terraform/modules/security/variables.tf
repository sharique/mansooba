variable "vpc_id" {
  description = "ID of the VPC to place security groups in."
  type        = string
}

variable "allowed_ssh_cidr" {
  description = "CIDR block allowed to SSH to the EC2 instance. Restrict to your IP for better security."
  type        = string
  default     = "0.0.0.0/0"
}

variable "name_prefix" {
  description = "Prefix applied to all resource Name tags."
  type        = string
  default     = "mansooba"
}

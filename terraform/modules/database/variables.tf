variable "private_subnet_ids" {
  description = "List of private subnet IDs for the RDS subnet group (must span at least 2 AZs)."
  type        = list(string)
}

variable "security_group_id" {
  description = "ID of the RDS security group."
  type        = string
}

variable "db_password" {
  description = "Master password for the RDS PostgreSQL instance."
  type        = string
  sensitive   = true
}

variable "name_prefix" {
  description = "Prefix applied to all resource Name tags and identifiers."
  type        = string
  default     = "mansooba"
}

variable "db_name" {
  description = "Name of the initial database to create."
  type        = string
  default     = "mansooba"
}

variable "db_username" {
  description = "Master username for the RDS instance."
  type        = string
  default     = "mansooba"
}

variable "instance_class" {
  description = "RDS instance class."
  type        = string
  default     = "db.t3.micro"
}

variable "allocated_storage" {
  description = "Allocated storage for the RDS instance in GB."
  type        = number
  default     = 20
}

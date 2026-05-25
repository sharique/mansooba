variable "aws_region" {
  description = "AWS region — used to filter the Ubuntu AMI data source."
  type        = string
}

variable "subnet_id" {
  description = "ID of the public subnet to launch the EC2 instance into."
  type        = string
}

variable "security_group_id" {
  description = "ID of the EC2 security group."
  type        = string
}

variable "instance_profile_name" {
  description = "Name of the IAM instance profile to attach."
  type        = string
}

variable "ssh_public_key" {
  description = "SSH public key content to install on the instance (contents of ~/.ssh/mansooba.pub)."
  type        = string
}

variable "user_data" {
  description = "Pre-rendered EC2 user-data bootstrap script. Render with templatefile() in the root module."
  type        = string
}

variable "name_prefix" {
  description = "Prefix applied to all resource Name tags and the key pair name."
  type        = string
  default     = "mansooba"
}

variable "instance_type" {
  description = "EC2 instance type."
  type        = string
  default     = "t2.micro"
}

variable "root_volume_size_gb" {
  description = "Size of the root EBS volume in GB."
  type        = number
  default     = 20
}

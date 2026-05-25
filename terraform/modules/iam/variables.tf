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

variable "aws_region" {
  description = "AWS region the storage bucket is created in."
  type        = string
}

variable "name_prefix" {
  description = "Prefix applied to the bucket name."
  type        = string
  default     = "mansooba"
}

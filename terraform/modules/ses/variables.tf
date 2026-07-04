variable "smtp_from" {
  description = "Sender email address to verify with SES (e.g. noreply@yourdomain.com). AWS will send a verification email here — you must click the link before SES can send from this address."
  type        = string
}

variable "aws_region" {
  description = "AWS region the SES resources are created in. Determines the SMTP endpoint (email-smtp.<region>.amazonaws.com) and the ses_smtp_password_v4 signing key."
  type        = string
}

variable "name_prefix" {
  description = "Prefix applied to all IAM and SSM resource names."
  type        = string
  default     = "mansooba"
}

variable "ssm_path_prefix" {
  description = "SSM Parameter Store path prefix under which SMTP_USER and SMTP_PASS are stored (e.g. /mansooba)."
  type        = string
  default     = "/mansooba"
}

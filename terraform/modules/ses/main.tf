# ── SES Email Identity ────────────────────────────────────────────────────────
# Registers a sender address with AWS SES.
#
# After `terraform apply`, AWS sends a verification email to this address.
# You must click the link in that email before SES will send from it.
#
# Sandbox mode (default): SES can only send to verified addresses as well.
# To send to arbitrary users, request production access:
#   SES Console → Account dashboard → Request production access.

resource "aws_ses_email_identity" "sender" {
  email = var.smtp_from
}

# ── SES SMTP IAM User ─────────────────────────────────────────────────────────
# SES SMTP uses a dedicated IAM user — not the EC2 instance role.
# The SMTP password is derived from the IAM secret key using a signing
# algorithm (HMAC-SHA256). Terraform exposes this as ses_smtp_password_v4.
#
# Why a separate IAM user instead of letting the EC2 role send via SES?
# SMTP clients (like Go's net/smtp) authenticate with a username+password,
# not with SigV4. IAM roles cannot issue SMTP credentials.

resource "aws_iam_user" "ses_smtp" {
  name = "${var.name_prefix}-ses-smtp"
  tags = { Purpose = "SES SMTP authentication for the Mansooba backend email sender" }
}

# Allow this IAM user to send email via SES.
# ses:SendRawEmail is needed for multipart (HTML + text) messages;
# ses:SendEmail covers plain messages.
resource "aws_iam_user_policy" "ses_send" {
  name = "${var.name_prefix}-ses-send"
  user = aws_iam_user.ses_smtp.name
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["ses:SendRawEmail", "ses:SendEmail"]
      Resource = "*"
    }]
  })
}

# Create an access key for the SES SMTP IAM user.
# The raw secret key is NOT the SMTP password — ses_smtp_password_v4
# is the correctly derived credential that SMTP servers expect.
#
# ⚠ This access key is stored in Terraform state. Encrypt your state file
# (S3 backend with server-side encryption) in shared or production environments.
resource "aws_iam_access_key" "ses_smtp" {
  user = aws_iam_user.ses_smtp.name
}

# ── SMTP Credentials in SSM Parameter Store ───────────────────────────────────
# EC2 user-data fetches these at boot so credentials never appear in
# committed files, docker-compose env_file entries, or `docker inspect` output.
# Both are SecureString — encrypted at rest with the default KMS key.

resource "aws_ssm_parameter" "smtp_user" {
  name        = "${var.ssm_path_prefix}/SMTP_USER"
  description = "SES SMTP username (IAM access key ID for ${aws_iam_user.ses_smtp.name})"
  type        = "SecureString"
  value       = aws_iam_access_key.ses_smtp.id
}

resource "aws_ssm_parameter" "smtp_pass" {
  name        = "${var.ssm_path_prefix}/SMTP_PASS"
  description = "SES SMTP password (ses_smtp_password_v4 derived from IAM secret key, region-specific)"
  type        = "SecureString"
  value       = aws_iam_access_key.ses_smtp.ses_smtp_password_v4
}

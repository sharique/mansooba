output "smtp_host" {
  description = "SES SMTP endpoint for the configured region. Use as SMTP_HOST in the backend .env."
  value       = "email-smtp.${var.aws_region}.amazonaws.com"
}

output "smtp_port" {
  description = "SMTP port. 587 uses STARTTLS (recommended); 465 uses implicit TLS."
  value       = "587"
}

output "smtp_from" {
  description = "The verified sender address (same as var.smtp_from). Use as SMTP_FROM in the backend .env."
  value       = var.smtp_from
}

output "identity_arn" {
  description = "ARN of the SES email identity. Use to scope IAM send policies to this sender only."
  value       = aws_ses_email_identity.sender.arn
}

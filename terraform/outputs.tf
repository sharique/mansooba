output "ec2_public_ip" {
  description = "Auto-assigned public IP of the EC2 instance (no Elastic IP — see modules/compute/main.tf). Use this to SSH in and as the EC2_HOST GitHub Secret for CD; note it changes on stop/start."
  value       = module.compute.public_ip
}

output "rds_endpoint" {
  description = "RDS PostgreSQL hostname. Written automatically to SSM at /mansooba/RDS_ENDPOINT by aws_ssm_parameter.rds_endpoint — no manual step needed."
  value       = module.database.rds_endpoint
  sensitive   = true
}

output "ssh_command" {
  description = "Ready-to-run SSH command to connect to the EC2 instance."
  value       = module.compute.ssh_command
}

output "ses_smtp_host" {
  description = "SES SMTP endpoint written into the backend .env by user-data. Null when enable_ses is false."
  value       = var.enable_ses ? module.ses[0].smtp_host : null
}

output "ses_identity_arn" {
  description = "ARN of the SES email identity. Useful for scoping IAM send policies or debugging SES permissions. Null when enable_ses is false."
  value       = var.enable_ses ? module.ses[0].identity_arn : null
}

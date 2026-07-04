output "ec2_public_ip" {
  description = "Elastic IP of the EC2 instance. Use this to SSH in and as the EC2_HOST GitHub Secret for CD."
  value       = module.compute.public_ip
}

output "rds_endpoint" {
  description = "RDS PostgreSQL hostname. After apply, store this in SSM: aws ssm put-parameter --name /mansooba/RDS_ENDPOINT --value <value> --type String"
  value       = module.database.rds_endpoint
  sensitive   = true
}

output "ssh_command" {
  description = "Ready-to-run SSH command to connect to the EC2 instance."
  value       = module.compute.ssh_command
}

output "ses_smtp_host" {
  description = "SES SMTP endpoint written into the backend .env by user-data. For reference only."
  value       = module.ses.smtp_host
}

output "ses_identity_arn" {
  description = "ARN of the SES email identity. Useful for scoping IAM send policies or debugging SES permissions."
  value       = module.ses.identity_arn
}

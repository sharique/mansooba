output "ec2_public_ip" {
  description = "Elastic IP of the EC2 instance. Use this as EC2_HOST in GitHub Secrets."
  value       = module.compute.public_ip
}

output "rds_endpoint" {
  description = "RDS PostgreSQL endpoint hostname. Store in SSM as /mansooba/RDS_ENDPOINT."
  value       = module.database.rds_endpoint
  sensitive   = true
}

output "ssh_command" {
  description = "Command to SSH into the EC2 instance."
  value       = module.compute.ssh_command
}

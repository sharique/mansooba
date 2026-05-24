output "ec2_public_ip" {
  description = "Elastic IP of the EC2 instance. Use this as EC2_HOST in GitHub Secrets."
  value       = aws_eip.app.public_ip
}

output "rds_endpoint" {
  description = "RDS PostgreSQL endpoint hostname. Store in SSM as /mansooba/RDS_ENDPOINT."
  value       = aws_db_instance.postgres.address
  sensitive   = true
}

output "ssh_command" {
  description = "Command to SSH into the EC2 instance."
  value       = "ssh -i ~/.ssh/mansooba ubuntu@${aws_eip.app.public_ip}"
}

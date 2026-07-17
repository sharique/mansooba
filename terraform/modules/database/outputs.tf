output "rds_endpoint" {
  description = "Hostname of the RDS PostgreSQL instance. Store in SSM as /mansooba/RDS_ENDPOINT."
  value       = aws_db_instance.postgres.address
  sensitive   = true
}

output "rds_arn" {
  description = "ARN of the RDS PostgreSQL instance, used to scope the least-privilege IAM policy that lets the backend stop/start it (feature 010, db-idle-autostop)."
  value       = aws_db_instance.postgres.arn
}

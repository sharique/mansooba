output "rds_endpoint" {
  description = "Hostname of the RDS PostgreSQL instance. Store in SSM as /mansooba/RDS_ENDPOINT."
  value       = aws_db_instance.postgres.address
  sensitive   = true
}

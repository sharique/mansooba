output "rds_endpoint" {
  description = "Hostname of the RDS PostgreSQL instance. Store in SSM as /mansooba/RDS_ENDPOINT."
  value       = aws_db_instance.postgres.address
  sensitive   = true
}

output "rds_arn" {
  description = "ARN of the RDS PostgreSQL instance, used to scope the least-privilege IAM policy that lets the backend stop/start it (feature 010, db-idle-autostop)."
  value       = aws_db_instance.postgres.arn
}

output "rds_identifier" {
  description = "DB instance identifier (aws_db_instance.postgres.identifier), written to the backend's RDS_INSTANCE_IDENTIFIER env var so Config.RDSAutoStopApplies() can cross-validate it against DB_DSN's host (feature 010, db-idle-autostop)."
  value       = aws_db_instance.postgres.identifier
}

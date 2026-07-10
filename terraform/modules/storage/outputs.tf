output "bucket_name" {
  description = "Name of the attachments S3 bucket. Use as STORAGE_BUCKET in the backend .env."
  value       = aws_s3_bucket.attachments.id
}

output "bucket_arn" {
  description = "ARN of the attachments S3 bucket. Used to scope the EC2 IAM role's S3 policy to this bucket only."
  value       = aws_s3_bucket.attachments.arn
}

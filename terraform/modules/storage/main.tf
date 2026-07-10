# ── Attachment Storage Bucket ──────────────────────────────────────────────────
# Holds issue attachment files uploaded by the backend. The backend never
# receives static AWS credentials for this bucket — it authenticates via the
# EC2 instance's IAM role (see modules/iam), which is granted a scoped policy
# against this bucket's ARN.
#
# No public access, no static website hosting, no versioning (the product
# spec explicitly excludes file versioning — replacing a file means deleting
# the old attachment and uploading a new one, so keeping old S3 object
# versions around would only add unused storage cost).

resource "aws_s3_bucket" "attachments" {
  bucket = "${var.name_prefix}-attachments"
  tags   = { Purpose = "Issue attachment files for the Mansooba backend" }
}

resource "aws_s3_bucket_public_access_block" "attachments" {
  bucket = aws_s3_bucket.attachments.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Encrypt objects at rest with the default AWS-managed key (SSE-S3).
# No KMS key management needed for this use case's threat model.
resource "aws_s3_bucket_server_side_encryption_configuration" "attachments" {
  bucket = aws_s3_bucket.attachments.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

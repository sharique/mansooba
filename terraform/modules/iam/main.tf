# ── EC2 Instance Role ─────────────────────────────────────────────────────────
# An IAM role lets the EC2 instance call AWS APIs without embedding credentials.
# The trust policy here limits assumption to the EC2 service only —
# no human IAM user can assume this role directly.

resource "aws_iam_role" "ec2" {
  name = "${var.name_prefix}-ec2-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ec2.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

# ── SSM Read Policy ───────────────────────────────────────────────────────────
# Grants the EC2 instance read-only access to all SSM parameters under the
# /mansooba/* path prefix. This is how user-data.sh fetches secrets at boot:
#
#   aws ssm get-parameter --name /mansooba/JWT_SECRET --with-decryption
#
# Scoped to the specific path prefix and region to follow least-privilege.
# Covers all current parameters: JWT_SECRET, DB_PASSWORD, GHCR_PAT,
# RDS_ENDPOINT, SMTP_USER, SMTP_PASS.

resource "aws_iam_role_policy" "ssm_read" {
  name = "${var.name_prefix}-ssm-read"
  role = aws_iam_role.ec2.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "ssm:GetParameter",
        "ssm:GetParametersByPath"
      ]
      Resource = "arn:aws:ssm:${var.aws_region}:*:parameter${var.ssm_path_prefix}/*"
    }]
  })
}

# ── S3 Attachment Storage Policy ──────────────────────────────────────────────
# Grants the EC2 instance read/write/delete access to attachment objects in the
# storage bucket only — no s3:ListBucket, no wildcard resource. The backend
# never receives a static AWS access key for this; the SDK's default
# credential chain resolves this role automatically (research.md Decision 8
# in the docs repo's spec for the file-attachments feature).
#
# Resource is scoped to "<bucket_arn>/*" (objects), not the bucket ARN itself,
# since none of PutObject/GetObject/DeleteObject/DeleteObjects operate on the
# bucket resource — only on individual object keys within it.

resource "aws_iam_role_policy" "s3_attachments" {
  name = "${var.name_prefix}-s3-attachments"
  role = aws_iam_role.ec2.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "s3:PutObject",
        "s3:GetObject",
        "s3:DeleteObject",
        "s3:DeleteObjects"
      ]
      Resource = "${var.attachments_bucket_arn}/*"
    }]
  })
}

# ── Instance Profile ──────────────────────────────────────────────────────────
# An instance profile is the container that attaches an IAM role to an EC2
# instance. The EC2 launch API accepts an instance profile name, not a role ARN.

resource "aws_iam_instance_profile" "ec2" {
  name = "${var.name_prefix}-ec2-profile"
  role = aws_iam_role.ec2.name
}

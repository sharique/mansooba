# ── AMI: latest Ubuntu 24.04 LTS ─────────────────────────────────────────────
# Dynamically resolves the most recent Ubuntu 24.04 (Noble) AMI published by
# Canonical (owner ID 099720109477). This avoids hardcoding an AMI ID that
# would go stale as Canonical releases security patches.
#
# Note: the compute resource has `ignore_changes = [ami]` so that Terraform
# never replaces the running instance when a newer AMI is published.
# Run `terraform apply -replace=aws_instance.app` to trigger intentional replacement.

# Find latest linux Ami
data "aws_ami" "latest_linux" {
  most_recent = true
  owners      = ["amazon"]
  filter {
    name   = "name"
    values = ["al2023-ami-2023.*-x86_64"]
  }
}

# ── SSH Key Pair ──────────────────────────────────────────────────────────────
# Uploads the public half of your local SSH key to AWS so EC2 can install it
# in ubuntu's authorized_keys. The private key stays on your machine only.

resource "aws_key_pair" "deployer" {
  key_name   = "${var.name_prefix}-deployer"
  public_key = var.ssh_public_key
}

# ── EC2 Instance ──────────────────────────────────────────────────────────────
# t2.micro is within the free tier (750 hours/month for 12 months).
# The instance receives the IAM instance profile so it can read from SSM.
# user_data is the rendered bootstrap script from user-data.sh — it runs
# once on first boot as root and starts the Docker Compose stack.

resource "aws_instance" "app" {
  ami                    = data.aws_ami.latest_linux.id
  instance_type          = var.instance_type
  subnet_id              = var.subnet_id
  vpc_security_group_ids = [var.security_group_id]
  iam_instance_profile   = var.instance_profile_name
  key_name               = aws_key_pair.deployer.key_name
  user_data              = var.user_data

  root_block_device {
    volume_size = var.root_volume_size_gb
    volume_type = "gp3"
  }

  tags = { Name = "${var.name_prefix}-app" }

  lifecycle {
    # Prevent instance replacement when Canonical releases a new Ubuntu AMI or
    # when the user-data script changes. To force a replacement (e.g. to reprovision
    # from scratch), run: terraform apply -replace=aws_instance.app
    ignore_changes = [ami, user_data]
  }
}

# ── Elastic IP ────────────────────────────────────────────────────────────────
# Gives the EC2 instance a static public IP that survives stop/start cycles.
# Without an EIP, AWS assigns a new IP every time the instance restarts,
# which would break DNS records and GitHub Secrets pointing to the host.
# An EIP attached to a running instance is free; charges apply if it's
# allocated but not attached.

resource "aws_eip" "app" {
  instance = aws_instance.app.id
  domain   = "vpc"
  tags     = { Name = "${var.name_prefix}-eip" }
}

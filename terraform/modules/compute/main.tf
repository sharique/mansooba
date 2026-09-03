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
  ami                         = data.aws_ami.latest_linux.id
  instance_type               = var.instance_type
  subnet_id                   = var.subnet_id
  vpc_security_group_ids      = [var.security_group_id]
  iam_instance_profile        = var.instance_profile_name
  key_name                    = aws_key_pair.deployer.key_name
  user_data                   = var.user_data
  associate_public_ip_address = true

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

# ── Public IP ─────────────────────────────────────────────────────────────────
# No Elastic IP here on purpose: some AWS accounts (notably org-managed ones,
# e.g. under a bootcamp/school AWS Organization) have an SCP that explicitly
# denies ec2:AllocateAddress, which fails `terraform apply` on `aws_eip`
# with an UnauthorizedOperation error that no IAM permission — not even
# AdministratorAccess — can override.
#
# Instead the instance gets AWS's normal auto-assigned public IP
# (associate_public_ip_address above + map_public_ip_on_launch on the public
# subnet, see modules/networking). Trade-off: that IP changes on every
# stop/start, unlike an EIP. If your account is allowed to allocate EIPs,
# see terraform/README.md for how to add one back.

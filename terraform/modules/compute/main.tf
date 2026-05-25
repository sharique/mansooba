# ── AMI: latest Ubuntu 24.04 LTS ─────────────────────────────────────────────

data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"] # Canonical

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-noble-24.04-amd64-server-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

# ── SSH Key Pair ──────────────────────────────────────────────────────────────

resource "aws_key_pair" "deployer" {
  key_name   = "${var.name_prefix}-deployer"
  public_key = var.ssh_public_key
}

# ── EC2 Instance ──────────────────────────────────────────────────────────────

resource "aws_instance" "app" {
  ami                    = data.aws_ami.ubuntu.id
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
    # Prevent Terraform from replacing the instance when the AMI is updated.
    # Run 'terraform apply -replace=aws_instance.app' to trigger intentional replacement.
    ignore_changes = [ami, user_data]
  }
}

# ── Elastic IP ────────────────────────────────────────────────────────────────

resource "aws_eip" "app" {
  instance = aws_instance.app.id
  domain   = "vpc"
  tags     = { Name = "${var.name_prefix}-eip" }
}

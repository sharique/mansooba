# ── VPC ───────────────────────────────────────────────────────────────────────
# A dedicated VPC isolates Mansooba from any other resources in the account.
# DNS support and hostnames are enabled so RDS generates a resolvable endpoint
# (e.g. mansooba-db.cxxx.us-east-1.rds.amazonaws.com) rather than just an IP.

resource "aws_vpc" "main" {
  cidr_block           = var.vpc_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true
  tags                 = { Name = "${var.name_prefix}-vpc" }
}

# ── Internet Gateway ──────────────────────────────────────────────────────────
# Connects the VPC to the public internet. Required for:
#   • EC2 to pull Docker images from GHCR
#   • EC2 to send email via SES SMTP (outbound port 587)
#   • Users to reach the frontend and backend API

resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id
  tags   = { Name = "${var.name_prefix}-igw" }
}

# ── Public Subnet ─────────────────────────────────────────────────────────────
# EC2 lives here. map_public_ip_on_launch gives the instance an initial public IP
# (though the Elastic IP in the compute module is what we actually use).

resource "aws_subnet" "public" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = var.public_subnet_cidr
  availability_zone       = "${var.aws_region}a"
  map_public_ip_on_launch = true
  tags                    = { Name = "${var.name_prefix}-public" }
}

# ── Private Subnets ───────────────────────────────────────────────────────────
# RDS lives in private subnets — not reachable from the internet.
# Two subnets in different AZs are required by the RDS subnet group even for a
# single-AZ instance (AWS enforces this for failover readiness).

resource "aws_subnet" "private_a" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = var.private_subnet_a_cidr
  availability_zone = "${var.aws_region}a"
  tags              = { Name = "${var.name_prefix}-private-a" }
}

resource "aws_subnet" "private_b" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = var.private_subnet_b_cidr
  availability_zone = "${var.aws_region}b"
  tags              = { Name = "${var.name_prefix}-private-b" }
}

# ── Public Route Table ────────────────────────────────────────────────────────
# Routes all outbound traffic (0.0.0.0/0) from the public subnet through the
# internet gateway. Private subnets have no route table association so their
# instances have no internet route (and no outbound internet access).

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }
  tags = { Name = "${var.name_prefix}-rt-public" }
}

resource "aws_route_table_association" "public" {
  subnet_id      = aws_subnet.public.id
  route_table_id = aws_route_table.public.id
}

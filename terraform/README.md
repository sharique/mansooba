# Mansooba Terraform — AWS Infrastructure

Provisions: VPC, public/private subnets, EC2 t2.micro (auto-assigned public IP — no Elastic IP by default, see below), RDS PostgreSQL db.t3.micro, IAM, security groups.

> Quick reference — for the full walkthrough (with troubleshooting for
> real-world snags like pre-existing resources or expired SSO sessions), see
> [docs/deployments/deploy-to-aws-terraform.md](../docs/deployments/deploy-to-aws-terraform.md).

**Free tier eligible** — EC2 t2.micro + RDS db.t3.micro are free for 12 months on a new AWS account.

## Prerequisites

1. AWS CLI configured: `aws configure`
2. Terraform installed: `brew install terraform`
3. SSH key pair: `ssh-keygen -t ed25519 -f ~/.ssh/mansooba`

## First-time setup

> No GitHub PAT needed — the GHCR images are public, so `user-data.sh` pulls
> them without authenticating.
>
> No manual SSM step needed either: `JWT_SECRET` is generated and
> `DB_PASSWORD`/`RDS_ENDPOINT` are written to SSM automatically by
> `aws_ssm_parameter.jwt_secret` / `.db_password` / `.rds_endpoint` in
> `main.tf` — `module.compute` explicitly depends on all three, so
> `terraform apply` always creates them before the instance boots.

### 1. Configure variables

```bash
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars:
# - aws_region       (default: eu-central-1)
# - ssh_public_key    (cat ~/.ssh/mansooba.pub)
# - db_password       (Terraform writes this to SSM for you)
```

By default this also provisions SES (email identity + SMTP IAM user) for
password-reset emails. To skip email entirely — the backend falls back to
returning the reset token directly in the API response instead — set
`enable_ses = false` in `terraform.tfvars` and drop `smtp_from` (unused when
disabled).

RDS idle auto-stop/wake-on-hit (feature 010, ADR-030) is also configurable —
`rds_autostop_enabled`, `rds_idle_timeout`, `rds_idle_check_interval`, and
`rds_start_failure_bound` in `terraform.tfvars` — see `terraform.tfvars.example`
for the defaults, which match the backend's own built-in behavior.

### 2. Init and apply

```bash
cd terraform
terraform init
terraform plan
terraform apply
```

### 3. SSH in and verify

```bash
terraform output ssh_command | bash
# Inside EC2:
docker ps                          # should show backend + frontend
curl http://localhost:8080/health  # {"status":"ok","db":"ok"}
```

## Adding a static IP back (Elastic IP)

By default `modules/compute` gives the EC2 instance AWS's normal
auto-assigned public IP, not an Elastic IP — it changes every time the
instance stops and starts. This is deliberate: some AWS accounts (e.g.
inside an org-managed AWS Organization) have a Service Control Policy that
explicitly denies `ec2:AllocateAddress`, which fails `terraform apply` with
an `UnauthorizedOperation` error no IAM permission can override. See
[docs/deployments/deploy-to-aws-terraform.md](../docs/deployments/deploy-to-aws-terraform.md#elastic-ip-allocation-denied-by-an-scp)
for how to confirm that's actually your situation.

If your account **is** allowed to allocate Elastic IPs and you want a stable
IP again (e.g. for DNS records or a GitHub Secret that shouldn't change),
add this to `modules/compute/main.tf`:

```hcl
resource "aws_eip" "app" {
  instance = aws_instance.app.id
  domain   = "vpc"
  tags     = { Name = "${var.name_prefix}-eip" }
}
```

Then point both outputs in `modules/compute/outputs.tf` at it instead of the
instance's own IP:

```hcl
output "public_ip" {
  description = "Elastic IP address of the EC2 instance."
  value       = aws_eip.app.public_ip
}

output "ssh_command" {
  description = "Command to SSH into the EC2 instance."
  value       = "ssh -i ~/.ssh/mansooba ec2-user@${aws_eip.app.public_ip}"
}
```

Run `terraform apply` again — it allocates and associates the EIP without
touching the running instance.

## Upgrading to S3 backend (optional, for teams)

```bash
# Create S3 bucket and DynamoDB lock table
# (LocationConstraint is required for any region other than us-east-1 —
# omitting it here would create the bucket in us-east-1 regardless of
# --region, then fail with IllegalLocationConstraintException.)
aws s3api create-bucket --bucket mansooba-tf-state --region eu-central-1 \
  --create-bucket-configuration LocationConstraint=eu-central-1
aws s3api put-bucket-versioning --bucket mansooba-tf-state \
  --versioning-configuration Status=Enabled
aws dynamodb create-table --table-name mansooba-tf-lock \
  --attribute-definitions AttributeName=LockID,AttributeType=S \
  --key-schema AttributeName=LockID,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST --region eu-central-1

# Then add this to the terraform{} block in main.tf:
# backend "s3" {
#   bucket         = "mansooba-tf-state"
#   key            = "prod/terraform.tfstate"
#   region         = "eu-central-1"
#   dynamodb_table = "mansooba-tf-lock"
# }

terraform init -migrate-state
```

## Tear down

```bash
terraform destroy
```

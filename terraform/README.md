# Mansooba Terraform — AWS Infrastructure

Provisions: VPC, public/private subnets, EC2 t2.micro, RDS PostgreSQL db.t3.micro, IAM, security groups, Elastic IP.

**Free tier eligible** — EC2 t2.micro + RDS db.t3.micro are free for 12 months on a new AWS account.

## Prerequisites

1. AWS CLI configured: `aws configure`
2. Terraform installed: `brew install terraform`
3. SSH key pair: `ssh-keygen -t ed25519 -f ~/.ssh/mansooba`

## First-time setup

### 1. Store secrets in SSM Parameter Store

```bash
aws ssm put-parameter --name /mansooba/JWT_SECRET \
  --value "your-strong-jwt-secret" \
  --type SecureString --region us-east-1

aws ssm put-parameter --name /mansooba/DB_PASSWORD \
  --value "your-db-password" \
  --type SecureString --region us-east-1

# GitHub PAT with read:packages scope (create at github.com/settings/tokens)
aws ssm put-parameter --name /mansooba/GHCR_PAT \
  --value "ghp_xxxxxxxxxxxx" \
  --type SecureString --region us-east-1
```

### 2. Configure variables

```bash
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars:
# - aws_region
# - ssh_public_key (cat ~/.ssh/mansooba.pub)
# - db_password (must match the SSM value)
```

### 3. Init and apply

```bash
cd terraform
terraform init
terraform plan
terraform apply
```

### 4. Store the RDS endpoint in SSM

After `terraform apply`, store the RDS endpoint so user-data.sh can read it:

```bash
RDS_ENDPOINT=$(terraform output -raw rds_endpoint)
aws ssm put-parameter --name /mansooba/RDS_ENDPOINT \
  --value "$RDS_ENDPOINT" \
  --type String --region us-east-1
```

### 5. SSH in and verify

```bash
terraform output ssh_command | bash
# Inside EC2:
docker ps                          # should show backend + frontend
curl http://localhost:8080/health  # {"status":"ok","db":"ok"}
```

## Upgrading to S3 backend (optional, for teams)

```bash
# Create S3 bucket and DynamoDB lock table
aws s3api create-bucket --bucket mansooba-tf-state --region us-east-1
aws s3api put-bucket-versioning --bucket mansooba-tf-state \
  --versioning-configuration Status=Enabled
aws dynamodb create-table --table-name mansooba-tf-lock \
  --attribute-definitions AttributeName=LockID,AttributeType=S \
  --key-schema AttributeName=LockID,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST --region us-east-1

# Then add this to the terraform{} block in main.tf:
# backend "s3" {
#   bucket         = "mansooba-tf-state"
#   key            = "prod/terraform.tfstate"
#   region         = "us-east-1"
#   dynamodb_table = "mansooba-tf-lock"
# }

terraform init -migrate-state
```

## Tear down

```bash
terraform destroy
```

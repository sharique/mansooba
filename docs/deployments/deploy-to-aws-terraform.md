# Deploy Mansooba to AWS — Terraform Guide

> **Who this is for:** You're comfortable with the command line and want Infrastructure-as-Code instead of clicking through the AWS Console. This guide assumes you already know *what* SES/RDS/S3/IAM are — see [deploy-to-aws-beginner.md](deploy-to-aws-beginner.md) if you want those concepts explained step by step; the console-only guide and this one provision an equivalent stack.
>
> **Time:** ~20 minutes hands-on (`terraform apply` itself takes ~8–10 minutes, mostly waiting on RDS)
>
> **Cost:** $0 for the first 12 months on a new AWS account (EC2 t2.micro + RDS db.t3.micro are both free-tier eligible)

---

## Contents

- [What you'll build](#what-youll-build)
- [Prerequisites](#prerequisites)
- [Step 1 — Install Terraform and the AWS CLI](#step-1--install-terraform-and-the-aws-cli)
- [Step 2 — Generate an SSH key pair](#step-2--generate-an-ssh-key-pair)
- [Step 3 — Configure terraform.tfvars](#step-3--configure-terraformtfvars)
- [Step 4 — Init, plan, and apply](#step-4--init-plan-and-apply)
- [Step 5 — Verify](#step-5--verify)
- [Day-to-day operations](#day-to-day-operations)
  - [Update to a new version](#update-to-a-new-version)
  - [View logs](#view-logs)
  - [Change configuration](#change-configuration)
  - [Restart the app](#restart-the-app)
- [Tear down](#tear-down)
- [Troubleshooting](#troubleshooting)
  - [AWS says a resource already exists](#aws-says-a-resource-already-exists)
  - [AWS credentials expired or missing](#aws-credentials-expired-or-missing)
  - [Plan wants to replace the SSH key pair](#plan-wants-to-replace-the-ssh-key-pair)
  - [SES verification email never arrives](#ses-verification-email-never-arrives)
  - [App doesn't load after apply finishes](#app-doesnt-load-after-apply-finishes)
  - [Elastic IP allocation denied by an SCP](#elastic-ip-allocation-denied-by-an-scp)
- [Quick reference](#quick-reference)

---

## What you'll build

```
Your terminal (terraform apply)
     │
     ▼
EC2 instance  ──  a virtual computer in the cloud running Mansooba
  ├── Frontend (the web app UI on port 80)
  ├── Backend  (the API on port 8080)
  ├── Loki + Alloy  (System Logs audit trail + container log collection — always on)
  └── Grafana  (optional — off by default, see Day-to-day operations)
         │
         ├── RDS  ──  a managed PostgreSQL database (private, no internet access)
         ├── SES  ──  AWS Simple Email Service (optional — see enable_ses below)
         └── S3   ──  file storage for issue attachments (private, no internet access)
```

Everything is defined in `terraform/*.tf` — reading `main.tf` top to bottom is the
fastest way to see exactly what gets created and why (each module has a comment
block explaining its purpose).

> **No Elastic IP by default.** The EC2 instance gets AWS's normal
> auto-assigned public IP, which changes on stop/start. This is deliberate —
> some AWS accounts (org-managed ones especially) have a policy that blocks
> Elastic IP allocation entirely, and `terraform apply` would fail on it
> otherwise. See [terraform/README.md](../../terraform/README.md#adding-a-static-ip-back-elastic-ip)
> if you want one back and your account allows it.

---

## Prerequisites

- An **AWS account** — [aws.amazon.com](https://aws.amazon.com) (free tier covers everything here)
- **AWS CLI** installed and configured: `aws configure` (or `aws sso login` if your org uses SSO)
- **Terraform** ≥ 1.7 installed — `brew install terraform` (Mac) or see [terraform.io/downloads](https://developer.hashicorp.com/terraform/install)

---

## Step 1 — Install Terraform and the AWS CLI

```bash
terraform -version   # must be >= 1.7
aws --version
aws sts get-caller-identity   # confirms your credentials actually work
```

If the last command errors, fix your AWS credentials before continuing —
nothing past this point will work without them.

---

## Step 2 — Generate an SSH key pair

```bash
ssh-keygen -t ed25519 -f ~/.ssh/mansooba
```

Terraform uploads the **public** half (`~/.ssh/mansooba.pub`) to AWS; the
private half never leaves your machine.

---

## Step 3 — Configure terraform.tfvars

```bash
cd terraform
cp terraform.tfvars.example terraform.tfvars
```

Edit `terraform.tfvars`:

```hcl
aws_region     = "eu-central-1"
ssh_public_key = "ssh-ed25519 AAAA... your-key-comment"   # cat ~/.ssh/mansooba.pub
db_password    = "your-db-password"                        # Terraform writes this to SSM for you (see Step 4)
smtp_from      = "noreply@yourdomain.com"                  # verified SES sender; AWS emails you a verification link
```

> No GitHub PAT needed — the GHCR images are public, so the EC2 boot script
> pulls them without authenticating.
>
> No separate `aws ssm put-parameter` step exists for `JWT_SECRET` or
> `DB_PASSWORD` — `main.tf` generates `JWT_SECRET` (`random_password.jwt_secret`)
> and writes both, plus the RDS endpoint once it exists, to SSM automatically
> (`aws_ssm_parameter.jwt_secret` / `.db_password` / `.rds_endpoint`).
> `module.compute` explicitly depends on all three, so they always exist
> before the instance boots and fetches them — see the "Secrets in SSM
> Parameter Store" section near the bottom of `main.tf`.

Two things worth deciding now rather than after `apply`:

**Don't want email at all?** Set `enable_ses = false`. No SES email identity,
IAM user, or SMTP credentials get created at all — the backend falls back to
returning password-reset tokens directly in the API response. `smtp_from`
becomes unused in that case (leave it or delete it).

**Want different RDS idle auto-stop behavior?** `rds_autostop_enabled`,
`rds_idle_timeout`, `rds_idle_check_interval`, and `rds_start_failure_bound`
are all in `terraform.tfvars.example`, commented out with the backend's own
defaults shown — uncomment and change only the ones you want to override.

---

## Step 4 — Init, plan, and apply

```bash
terraform init
terraform plan    # review what it's about to create — should be ~24-29 resources
terraform apply   # type "yes" to confirm
```

`apply` takes about 8–10 minutes, almost entirely waiting for RDS to finish
provisioning. Terraform prints progress as each resource completes, and by
the time the instance launches, `JWT_SECRET`, `DB_PASSWORD`, and
`RDS_ENDPOINT` are already in SSM for it to fetch at boot — no follow-up step.

> If this isn't a fresh AWS account — e.g. you previously created any of this
> infrastructure by hand — `apply` will fail with `AlreadyExists` errors
> instead of quietly overwriting anything. See
> [Troubleshooting](#aws-says-a-resource-already-exists) before proceeding.

---

## Step 5 — Verify

```bash
terraform output ssh_command | bash
# Inside EC2:
docker ps                                    # backend, frontend, loki, alloy all healthy
curl http://localhost:8080/health
# {"status":"ok","db":"ok","db_latency_ms":...,"loki":"ok"}
```

Then open `http://<ec2_public_ip>` in your browser (get the IP with
`terraform output ec2_public_ip`) and run through the first-run setup wizard.

---

## Day-to-day operations

### Update to a new version

```bash
terraform output ssh_command | bash
# Inside EC2:
cd /opt/mansooba
sudo docker compose -f compose.prod.yml pull
sudo docker compose -f compose.prod.yml up -d --remove-orphans
sudo docker image prune -f
```

### View logs

```bash
# Raw container output
sudo docker logs mansooba-backend --tail 50
```

For security/admin events specifically, sign in to the app as an admin and
open **System → Logs** — that's the curated audit trail (Loki-backed), not
raw container output. For everything across every container in one place,
turn on Grafana:

```bash
sudo docker compose -f compose.prod.yml --profile observability up -d grafana
```

Grafana's port isn't opened to the internet on purpose (see `modules/security`)
— reach it through an SSH tunnel instead:

```bash
ssh -i ~/.ssh/mansooba -L 3001:localhost:3001 ec2-user@<ec2_public_ip>
```

Then open `http://localhost:3001` — `GF_SECURITY_ADMIN_USER` /
`GF_SECURITY_ADMIN_PASSWORD` come from `.env` on the instance (or set your
own via SSM `/mansooba/GRAFANA_ADMIN_PASSWORD` before the *first* boot — see
`terraform/user-data.sh`'s comments for why it's a fallback-generated password
otherwise).

### Change configuration

Most `terraform.tfvars` changes (`enable_ses`, the RDS auto-stop knobs,
`allowed_ssh_cidr`) only take effect on the *next* `terraform apply` if they
change something Terraform manages directly — but the values that flow into
`.env` via the startup script are only written once, at first boot. To apply
a config change to an already-running instance without recreating it:

```bash
terraform plan   # confirm exactly what would change
terraform apply
# then, if the change was to a .env-only value (not an actual AWS resource):
terraform output ssh_command | bash
sudo docker compose -f /opt/mansooba/compose.prod.yml up -d --force-recreate backend
```

### Restart the app

```bash
terraform output ssh_command | bash
sudo docker compose -f /opt/mansooba/compose.prod.yml restart
```

---

## Tear down

```bash
terraform destroy
```

Free tier lasts 12 months from account creation, not from when you deployed
this — if you're past that window, `destroy` stops the ongoing EC2/RDS
charges. This deletes everything Terraform created, including the database
(`skip_final_snapshot = true` — no backup is taken; see `modules/database/main.tf`
if you want that changed before tearing down a database you actually care about).

---

## Troubleshooting

### AWS says a resource already exists

Terraform's state file doesn't know about a resource that already exists in
your AWS account — most commonly because it was created manually, by a
previous `apply` whose state file was lost, or by a colleague using a
different state backend. This is **not** Terraform silently overwriting
anything; `apply` refuses and stops.

Two ways forward, depending on what you actually want:

- **You want Terraform to manage the existing resource going forward:**
  `terraform import <resource address> <id>`, e.g.
  `terraform import module.database.aws_db_instance.postgres mansooba-db`.
  Run `terraform state list` to see the exact resource addresses Terraform
  expects. **Read the plan carefully after importing** before applying
  anything — imported RDS instances in particular can show a spurious
  "change" to the `password` field, since AWS never returns the real
  password value for Terraform to compare against.
- **You don't want to touch the existing resource at all** (e.g. it's a real
  production deployment and you just want to test the Terraform code
  itself): don't run `apply`. `terraform validate` and `terraform plan`
  (against a placeholder `-var` set) both fully exercise the HCL — parsing,
  type-checking, the whole resource graph — without creating, modifying, or
  even reading real state. Neither one can collide with anything.

### AWS credentials expired or missing

Your AWS credentials expired mid-session (common with SSO — sessions are
often only valid for a few hours). Re-authenticate and retry:

```bash
aws sso login   # or: aws configure, if using long-lived access keys
terraform plan  # confirm credentials work again before re-running apply
```

### Plan wants to replace the SSH key pair

`aws_key_pair` is keyed by its public key content — if `ssh_public_key` in
`terraform.tfvars` doesn't exactly match what's already deployed (e.g. you
regenerated the key, or pasted it with a trailing newline difference),
Terraform proposes replacing it. Confirm the value in `terraform.tfvars`
matches `cat ~/.ssh/mansooba.pub` exactly before applying — a real key change
here recreates the EC2 instance too (`aws_instance.app` references the key
pair), which is disruptive if you didn't mean to do that.

### SES verification email never arrives

Same cause and fix as the [console guide's password-reset
troubleshooting](deploy-to-aws-beginner.md#no-password-reset-email-arrives) —
check spam, confirm `smtp_from` in `terraform.tfvars` is spelled correctly,
and remember SES starts in sandbox mode (can only send *to* verified
addresses too).

### App doesn't load after apply finishes

Give it a minute — `user-data.sh` runs Docker installation, image pulls, and
the RDS wait sequence after the instance boots, not before. Then:

```bash
terraform output ssh_command | bash
sudo tail -100 /var/log/user-data.log   # the bootstrap script's own log
```

Look for the last line reached. `JWT_SECRET`/`DB_PASSWORD`/`RDS_ENDPOINT` are
now written to SSM automatically before the instance ever boots (Step 3), so
a `ParameterNotFound` error here almost always means one of the three
`aws_ssm_parameter` resources in `main.tf` was skipped or removed from state
— e.g. a targeted `apply -target=module.compute` that bypassed them, or the
parameter was deleted manually outside Terraform. Check with:

```bash
aws ssm get-parameter --name /mansooba/JWT_SECRET --region eu-central-1
aws ssm get-parameter --name /mansooba/DB_PASSWORD --region eu-central-1
aws ssm get-parameter --name /mansooba/RDS_ENDPOINT --region eu-central-1
```

then `terraform apply` (no `-target`) to recreate whichever is missing, and
`terraform apply -replace=module.compute.aws_instance.app` to force the
instance to re-run its bootstrap script now that the parameters exist —
`ignore_changes = [ami, user_data]` on `aws_instance.app` means a plain
`apply` won't touch an already-running instance.

### Elastic IP allocation denied by an SCP

```
Error: creating EC2 EIP: operation error EC2: AllocateAddress, https response
error StatusCode: 403 ... UnauthorizedOperation: You are not authorized to
perform this operation ... with an explicit deny in a service control policy
```

This build doesn't provision an Elastic IP at all (see [What you'll
build](#what-youll-build)) — the instance uses AWS's normal auto-assigned
public IP instead, specifically to avoid this error. If you're seeing it
anyway, you're on an older checkout; pull the latest `terraform/` changes.

If you added an `aws_eip` resource back yourself (see
[terraform/README.md](../../terraform/README.md#adding-a-static-ip-back-elastic-ip))
and now hit this, decode the failure to confirm the cause:

```bash
aws sts decode-authorization-message --encoded-message "<the long string after 'Encoded authorization failure message:'>"
```

Look for `"explicitDeny":true` and an SCP statement denying
`ec2:AllocateAddress` (often named something like `DenyElasticIPCreation`).
That's an AWS Organizations Service Control Policy attached above your
account — it overrides even `AdministratorAccess`, and no IAM change on your
end can fix it. This is common on org-managed accounts (school/bootcamp AWS
Organizations in particular) that block EIPs to prevent idle-IP charges.
Your only path forward is asking whoever manages the AWS Organization for an
exception, or dropping the Elastic IP and accepting a public IP that changes
on stop/start (the default here).

---

## Quick reference

| Command | What it does |
|---|---|
| `terraform plan` | Shows what would change — always safe, never modifies anything |
| `terraform apply` | Creates/updates real AWS resources |
| `terraform output ssh_command \| bash` | SSHes into the EC2 instance |
| `terraform output ec2_public_ip` | The app's public IP |
| `terraform state list` | Every resource Terraform currently tracks |
| `terraform destroy` | Deletes everything Terraform created |
| `terraform validate` | Checks the HCL is well-formed — no AWS calls, no credentials needed |

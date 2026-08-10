# Deploy Mansooba to AWS — Beginner Guide (Console Only)

> **Who this is for:** You've never used AWS before. Every step uses the AWS web console — point and click. The only text editing is filling in a template file before you paste it into one field.
>
> **Time:** ~90 minutes (the database takes 10 minutes to start on its own — great time for a coffee break)
>
> **Cost:** $0 for the first 12 months on a new AWS account (free tier covers everything here)

---

## Contents

- [What you'll build](#what-youll-build)
- [Before you start — collect these things](#before-you-start-collect-these-things)
- [Step 1 — Create your AWS account](#step-1-create-your-aws-account)
- [Step 2 — Set up email with AWS SES](#step-2-set-up-email-with-aws-ses)
  - [2.1 Verify your sender email address](#21-verify-your-sender-email-address)
  - [2.2 Create SMTP credentials](#22-create-smtp-credentials)
  - [2.3 (Optional) Request production access](#23-optional-request-production-access)
- [Step 3 — Set up firewall rules (Security Groups)](#step-3-set-up-firewall-rules-security-groups)
  - [3.1 Open the Security Groups page](#31-open-the-security-groups-page)
  - [3.2 Create the web server security group](#32-create-the-web-server-security-group)
  - [3.3 Create the database security group](#33-create-the-database-security-group)
- [Step 4 — Create the database (RDS)](#step-4-create-the-database-rds)
- [Step 5 — Create an SSH key pair (to log into your server)](#step-5-create-an-ssh-key-pair-to-log-into-your-server)
- [Step 6 — Create an S3 bucket and IAM Role for EC2](#step-6-create-an-s3-bucket-and-iam-role-for-ec2)
  - [6.1 Create the attachments bucket](#61-create-the-attachments-bucket)
  - [6.2 Create the IAM role](#62-create-the-iam-role)
  - [6.3 Grant the role access to the attachments bucket](#63-grant-the-role-access-to-the-attachments-bucket)
  - [6.4 Grant the role permission to stop and start the database](#64-grant-the-role-permission-to-stop-and-start-the-database)
- [Step 7 — Wait for RDS to finish, then copy its address](#step-7-wait-for-rds-to-finish-then-copy-its-address)
- [Step 8 — Fill in your startup script template](#step-8-fill-in-your-startup-script-template)
- [Step 9 — Launch your EC2 server](#step-9-launch-your-ec2-server)
- [Step 10 — Assign a permanent IP address](#step-10-assign-a-permanent-ip-address)
- [Step 11 — Update the app URL in the configuration](#step-11-update-the-app-url-in-the-configuration)
- [Step 12 — Wait for the app to start](#step-12-wait-for-the-app-to-start)
- [Step 13 — Open the app](#step-13-open-the-app)
- [Day-to-day operations](#day-to-day-operations)
  - [Update to a new version](#update-to-a-new-version)
  - [View the application logs](#view-the-application-logs)
  - [Restart the app](#restart-the-app)
- [Tear down (when you're done)](#tear-down-when-youre-done)
- [Troubleshooting](#troubleshooting)
  - [The app doesn't load in the browser](#the-app-doesnt-load-in-the-browser)
  - [The database shows an error](#the-database-shows-an-error)
  - [No password-reset email arrives](#no-password-reset-email-arrives)
  - [Attaching a file to an issue fails, or the download link doesn't work](#attaching-a-file-to-an-issue-fails-or-the-download-link-doesnt-work)
  - [The database never seems to stop, or logs show "db idle auto-stop disabled"](#the-database-never-seems-to-stop-or-logs-show-db-idle-auto-stop-disabled)
  - [RDS shows "Stopped" in the console, or the app briefly shows a "waking up" message](#rds-shows-stopped-in-the-console-or-the-app-briefly-shows-a-waking-up-message)
- [Quick reference — what everything is](#quick-reference-what-everything-is)

---

## What you'll build

```
Your browser
     │
     ▼
EC2 instance  ──  a virtual computer in the cloud running Mansooba
  ├── Frontend (the web app UI on port 80)
  └── Backend  (the API on port 8080)
         │
         ├── RDS  ──  a managed PostgreSQL database (private, no internet access)
         ├── SES  ──  AWS Simple Email Service (3,000 free emails/month)
         └── S3   ──  file storage for issue attachments (private, no internet access)
```

---

## Before you start — collect these things

You will need:
- [ ] An **AWS account** — sign up at [aws.amazon.com](https://aws.amazon.com) (credit card required, but free tier means no charge)
- [ ] A **plain text editor** on your computer — Notepad on Windows, TextEdit on Mac (set to plain text: Format → Make Plain Text), or any code editor

---

## Step 1 — Create your AWS account

1. Go to [aws.amazon.com](https://aws.amazon.com) → click **Create an AWS Account**
2. Enter your email, choose a password, pick an account name (e.g. `my-mansooba`)
3. Enter your credit card (won't be charged within free tier)
4. Pick **Basic support** (free) → finish sign-up
5. Log in at [console.aws.amazon.com](https://console.aws.amazon.com)

**Set a billing alert so you're never surprised by a charge:**
1. In the top search bar type `Billing` → open **Billing and Cost Management**
2. In the left sidebar click **Budgets** → **Create budget**
3. Choose **Zero spend budget**
4. Enter your email address → **Create budget**

> You'll get an email the moment any charge appears. This catches accidental free-tier overruns early.

---

## Step 2 — Set up email with AWS SES

**SES** (Simple Email Service) is AWS's email service. It's free for up to 3,000 emails per month when sent from an EC2 server — no credit card charge, no external account needed.

SES starts in **sandbox mode**, meaning it can only send email *to* addresses you've verified. This is fine for testing. You can request to leave sandbox mode later (Step 2.3) when you're ready to send to real users.

> **Don't want email at all?** Skip this entire step. In Step 8's startup
> script, leave `SMTP_HOST`, `SMTP_FROM`, `SMTP_USER`, and `SMTP_PASS` blank
> instead of filling them in. The backend detects the empty `SMTP_HOST` and
> falls back to returning password-reset tokens directly in the API response
> instead of emailing them — the same behavior the Terraform deployment gets
> from setting `enable_ses = false`.

### 2.1 Verify your sender email address

This is the "From" address that will appear in password-reset emails.

1. In the AWS Console search bar type `SES` → open **Amazon Simple Email Service**
2. Make sure your region (top-right corner) is **US East (N. Virginia)** — `us-east-1`
3. In the left sidebar click **Verified identities** → **Create identity**
4. Identity type: **Email address**
5. Email address: enter your email (e.g. `noreply@yourdomain.com` or any email you own)
6. Click **Create identity**
7. Check your inbox — AWS sends a verification email with a link
8. Click the link to confirm

The identity status changes to **Verified** (may take a minute — reload the page).

**Write this down:**
```
SES sender email:   ________________________________
```

> In sandbox mode you must also verify any email address you want to *receive* test emails. Repeat steps 3–8 for your personal inbox.

### 2.2 Create SMTP credentials

SES provides an SMTP server you can point the backend at. AWS can generate the credentials for you automatically.

1. In the SES left sidebar click **SMTP settings**
2. You'll see the SMTP server hostname — write it down:
   ```
   SMTP hostname:   email-smtp.us-east-1.amazonaws.com
   ```
3. Click **Create SMTP credentials**
4. IAM user name: leave as default (e.g. `ses-smtp-user.XXXX`) → **Create user**
5. A box appears with your **SMTP username** and **SMTP password** — **copy both now, they won't be shown again**

**Write these down:**
```
SMTP username:   ________________________________
SMTP password:   ________________________________
```

### 2.3 (Optional) Request production access

In sandbox mode, SES only sends to verified addresses. When you're ready to send to real users:

1. In the SES left sidebar click **Account dashboard**
2. Under **Sending limits** click **Request production access**
3. Fill in the form — describe your use case (password reset emails for opt-in users)
4. AWS usually approves within 24 hours

You can skip this now and come back after testing.

---

## Step 3 — Set up firewall rules (Security Groups)

A **Security Group** is AWS's firewall. You'll create two:
- One for the web server — allows web traffic from the internet
- One for the database — allows traffic from the web server only (not the internet)

### 3.1 Open the Security Groups page

1. In the AWS Console search bar type `EC2` → open **EC2**
2. In the left sidebar scroll down to **Network & Security** → click **Security Groups**
3. Make sure your region (top-right corner) is set to **US East (N. Virginia)** — `us-east-1`

### 3.2 Create the web server security group

1. Click **Create security group**
2. Fill in:
   - Security group name: `mansooba-ec2-sg`
   - Description: `Mansooba web server`
   - VPC: leave as the default (it says "default")
3. Under **Inbound rules** click **Add rule** three times and fill in:

   | Type | Protocol | Port | Source | Why |
   |------|----------|------|--------|-----|
   | HTTP | TCP | 80 | Anywhere-IPv4 (`0.0.0.0/0`) | Web browser traffic |
   | Custom TCP | TCP | 8080 | Anywhere-IPv4 (`0.0.0.0/0`) | API access |
   | SSH | TCP | 22 | Anywhere-IPv4 (`0.0.0.0/0`) | Log into server |

4. Leave **Outbound rules** unchanged (Allow All is fine)
5. Click **Create security group**

### 3.3 Create the database security group

1. Click **Create security group** again
2. Fill in:
   - Security group name: `mansooba-rds-sg`
   - Description: `Mansooba database`
   - VPC: leave as the default
3. Under **Inbound rules** click **Add rule** once:

   | Type | Protocol | Port | Source | Why |
   |------|----------|------|--------|-----|
   | PostgreSQL | TCP | 5432 | Custom → type `mansooba` and select **mansooba-ec2-sg** | Only the web server can reach the DB |

4. Click **Create security group**

---

## Step 4 — Create the database (RDS)

**RDS** is AWS's managed database. "Managed" means AWS handles backups and restarts — you just point the app at it.

1. In the search bar type `RDS` → open **RDS**
2. Click **Create database**
3. Choose **Standard create**
4. Engine: **PostgreSQL**
5. Engine Version: pick the latest **PostgreSQL 16.x**
6. Templates: click **Free tier** (this auto-selects free options)
7. Under **Settings**:
   - DB instance identifier: `mansooba-db`
   - Master username: `mansooba`
   - Master password: make up a strong password → **write it down**
   - Confirm password
8. Under **Connectivity**:
   - VPC: select the **default VPC**
   - Public access: **No**
   - VPC security group: click **Choose existing** → remove the `default` group → add `mansooba-rds-sg`
9. Expand **Additional configuration** at the bottom:
   - Initial database name: `mansooba`
10. Click **Create database**

**Write this down:**
```
RDS master password:   ________________________________
```

> Provisioning takes 5–10 minutes. A spinning indicator shows **Creating**. Continue to the next steps while you wait.

---

## Step 5 — Create an SSH key pair (to log into your server)

An SSH key is like a password for the server, but stored as a file. You create it in AWS and the private half downloads to your computer automatically.

1. In the EC2 left sidebar go to **Network & Security** → **Key Pairs**
2. Click **Create key pair**
3. Fill in:
   - Name: `mansooba-key`
   - Key pair type: **ED25519**
   - Private key file format: **.pem** (Mac/Linux) or **.ppk** (Windows, if you use PuTTY) — pick **.pem** if unsure
4. Click **Create key pair**

The private key file downloads automatically (e.g. `mansooba-key.pem`). **Keep this file safe — you can't download it again.**

Move it to a safe folder on your computer:
- Mac/Linux: move it to `~/.ssh/mansooba-key.pem`
- Windows: move it to `C:\Users\YourName\.ssh\mansooba-key.pem`

---

## Step 6 — Create an S3 bucket and IAM Role for EC2

Mansooba stores file attachments (things people attach to issues) in an S3 bucket rather than on the server itself. The server needs a **bucket** to store them in, and an **IAM Role** — a set of permissions — that lets it read and write to that bucket without you ever typing in a password for it.

### 6.1 Create the attachments bucket

1. In the search bar type `S3` → open **S3**
2. Click **Create bucket**
3. Bucket name: `mansooba-attachments`
   *(Bucket names must be globally unique across all AWS customers — if this is taken, try `mansooba-attachments-yourname` and remember to use that exact name in Step 8 below.)*
4. AWS Region: make sure it matches the region you've used everywhere else — **US East (N. Virginia)** `us-east-1`
5. Leave **Block all public access** checked (the default) — attachments should never be publicly reachable
6. Under **Bucket Versioning**, leave **Disable** selected
7. Under **Default encryption**, leave **Server-side encryption with Amazon S3 managed keys (SSE-S3)** selected
8. Click **Create bucket**

### 6.2 Create the IAM role

1. In the search bar type `IAM` → open **IAM**
2. In the left sidebar click **Roles** → **Create role**
3. Trusted entity type: **AWS service**
4. Use case: **EC2** → click **Next**
5. On the permissions page — search for `AmazonEC2ContainerRegistryReadOnly` and check it
   *(This lets the server pull container images. If you don't find it, you can skip adding a policy — just click Next.)*
6. Role name: `mansooba-ec2-role`
7. Click **Create role**

### 6.3 Grant the role access to the attachments bucket

The managed policy above only covers pulling container images — it doesn't grant any S3 access. Add a second, narrowly-scoped policy just for the attachments bucket:

1. Still in **IAM** → **Roles**, click `mansooba-ec2-role`
2. Click the **Permissions** tab → **Add permissions** → **Create inline policy**
3. Click the **JSON** tab and replace the contents with:
   ```json
   {
     "Version": "2012-10-17",
     "Statement": [
       {
         "Effect": "Allow",
         "Action": ["s3:PutObject", "s3:GetObject", "s3:DeleteObject", "s3:DeleteObjects"],
         "Resource": "arn:aws:s3:::mansooba-attachments/*"
       }
     ]
   }
   ```
   *(If you had to use a different bucket name in Step 6.1, replace `mansooba-attachments` here too.)*
4. Click **Next**
5. Policy name: `mansooba-s3-attachments`
6. Click **Create policy**

> This grants only upload, download, and delete on objects inside this one bucket — nothing else in your AWS account.

### 6.4 Grant the role permission to stop and start the database

Mansooba can automatically stop the RDS database after 10 minutes of no activity and start it again the moment it's needed, to save cost on an always-on demo. This needs one more narrowly-scoped policy, the same way as Step 6.3:

1. Still in **IAM** → **Roles** → `mansooba-ec2-role` → **Permissions** tab → **Add permissions** → **Create inline policy**
2. Click the **JSON** tab and replace the contents with:
   ```json
   {
     "Version": "2012-10-17",
     "Statement": [
       {
         "Effect": "Allow",
         "Action": ["rds:StartDBInstance", "rds:StopDBInstance", "rds:DescribeDBInstances"],
         "Resource": "arn:aws:rds:us-east-1:YOUR_ACCOUNT_ID:db:mansooba-db"
       }
     ]
   }
   ```
   *(Replace `YOUR_ACCOUNT_ID` with your 12-digit AWS account number — click your account name in the top-right corner of the console to find it. If you used a different region or DB instance identifier earlier, replace those too.)*
3. Click **Next**
4. Policy name: `mansooba-rds-lifecycle`
5. Click **Create policy**

> This grants only starting, stopping, and checking the status of this one database instance — nothing else in your AWS account. If you skip this step, the app still works fine, but the auto-stop/wake-on-hit feature won't be able to start the database back up (see the Troubleshooting section).

---

## Step 7 — Wait for RDS to finish, then copy its address

1. Go back to **RDS** → **Databases**
2. Click `mansooba-db`
3. Wait until **Status** shows **Available** (reload the page every minute or two)
4. Under **Connectivity & security** find the **Endpoint** field
5. Copy the endpoint — it looks like: `mansooba-db.cxxxxxxxx.us-east-1.rds.amazonaws.com`

**Write this down:**
```
RDS endpoint:   ________________________________
```

---

## Step 8 — Fill in your startup script template

Open your plain text editor. Copy the entire block below, paste it in, then replace every `FILL_IN_...` placeholder with your actual values.

```
#!/bin/bash
set -euo pipefail
exec > /var/log/user-data.log 2>&1

echo "=== Bootstrap started ==="

# ── Install Docker ──────────────────────────────────
dnf update -y
dnf install -y docker
systemctl enable docker
systemctl start docker
usermod -aG docker ec2-user

# ── Install Docker Compose plugin ───────────────────
mkdir -p /usr/local/lib/docker/cli-plugins
curl -SL "https://github.com/docker/compose/releases/download/v2.27.1/docker-compose-linux-x86_64" \
  -o /usr/local/lib/docker/cli-plugins/docker-compose
chmod +x /usr/local/lib/docker/cli-plugins/docker-compose

# ── Create app directory ─────────────────────────────
mkdir -p /opt/mansooba
cd /opt/mansooba

# ── Write configuration file ─────────────────────────
cat > .env << 'ENV'
SERVER_PORT=8080
JWT_SECRET=FILL_IN_RANDOM_SECRET
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h
LOG_LEVEL=info
APP_ENV=production

DB_DRIVER=postgres
DB_DSN=host=FILL_IN_RDS_ENDPOINT user=mansooba password=FILL_IN_DB_PASSWORD dbname=mansooba port=5432 sslmode=require

# Must match the DB instance identifier from Step 4 exactly — it has to be the
# leading label of the RDS endpoint above (mansooba-db.cxxxxxxxx...) for the
# auto-stop/wake-on-hit feature to recognize this is really that RDS instance.
RDS_INSTANCE_IDENTIFIER=mansooba-db

# Required for the RDS SDK client — without this, the app fails to start once
# auto-stop is enabled (the AWS SDK does not infer region from the EC2 instance
# automatically; only credentials come from the instance role).
AWS_REGION=us-east-1

# These four match the backend's own built-in defaults — spelled out here so
# they're visible/trackable instead of silently relying on whatever the
# backend happens to default to. Same variables the Terraform deployment
# exposes as rds_autostop_enabled/rds_idle_timeout/rds_idle_check_interval/
# rds_start_failure_bound.
RDS_AUTOSTOP_ENABLED=true
RDS_IDLE_TIMEOUT=10m
RDS_IDLE_CHECK_INTERVAL=1m
RDS_START_FAILURE_BOUND=3

# No access key/secret here — the server authenticates to S3 using the
# mansooba-ec2-role IAM role from Step 6, not a password.
STORAGE_BUCKET=mansooba-attachments
STORAGE_REGION=us-east-1

SMTP_HOST=email-smtp.us-east-1.amazonaws.com
SMTP_PORT=587
SMTP_FROM=FILL_IN_SES_SENDER_EMAIL
SMTP_USER=FILL_IN_SES_SMTP_USERNAME
SMTP_PASS=FILL_IN_SES_SMTP_PASSWORD

APP_BASE_URL=FILL_IN_LATER
CORS_ORIGINS=FILL_IN_LATER
REVOKED_TOKEN_CLEANUP_INTERVAL=15m

# System Logs — a durable audit trail (logins, admin actions, settings
# changes) backed by Grafana Loki, started automatically below. Recommended:
# it's what makes "who changed this setting, and when" answerable at all.
LOKI_BASE_URL=http://loki:3100
LOKI_RUNTIME_OVERRIDES_PATH=/etc/loki/runtime-overrides.yaml
LOKI_RETENTION_SYNC_INTERVAL=5m

# Grafana admin login — only used if you turn Grafana on (see the note after
# this script). Pick your own password; the placeholder below is just an
# example, not a requirement.
GF_SECURITY_ADMIN_USER=admin
GF_SECURITY_ADMIN_PASSWORD=FILL_IN_GRAFANA_PASSWORD
GF_AUTH_ANONYMOUS_ENABLED=false
ENV

# ── Fetch Loki, Alloy, and Grafana config from the repo ─────────────
# These are static config files (not secrets) — same ones the main
# Terraform-based deployment uses. Loki is the audit-log backing store;
# Alloy tails every container's logs into it; both start automatically.
# Grafana (a way to browse those logs) is included but off by default —
# see the note after this script for how to turn it on.
mkdir -p loki grafana/provisioning/datasources grafana/provisioning/dashboards alloy
cat > loki/runtime-overrides.yaml << 'EOF'
overrides: {}
EOF
for f in loki/local-config.yaml \
         grafana/provisioning/datasources/loki.yaml \
         grafana/provisioning/dashboards/dashboards.yaml \
         grafana/provisioning/dashboards/system-logs.json \
         grafana/provisioning/dashboards/container-logs.json \
         alloy/config.alloy; do
  curl -fsSL "https://raw.githubusercontent.com/sharique/mansooba/main/$f" -o "$f"
done

# ── Write Docker Compose file ────────────────────────
cat > compose.prod.yml << 'COMPOSE'
services:
  backend:
    image: ghcr.io/sharique/mansooba-backend:latest
    restart: unless-stopped
    env_file: .env
    ports:
      - "8080:8080"
    volumes:
      - ./loki/runtime-overrides.yaml:/etc/loki/runtime-overrides.yaml
    depends_on:
      loki:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/health"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 10s

  frontend:
    image: ghcr.io/sharique/mansooba-frontend:latest
    restart: unless-stopped
    ports:
      - "80:80"
    depends_on:
      backend:
        condition: service_healthy

  loki:
    image: grafana/loki:3.2.0
    restart: unless-stopped
    command: -config.file=/etc/loki/local-config.yaml -runtime-config.file=/etc/loki/runtime-overrides.yaml
    volumes:
      - loki_data:/loki
      - ./loki/local-config.yaml:/etc/loki/local-config.yaml
      - ./loki/runtime-overrides.yaml:/etc/loki/runtime-overrides.yaml
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:3100/ready"]
      interval: 15s
      timeout: 5s
      retries: 5
      start_period: 10s

  alloy:
    image: grafana/alloy:v1.4.3
    restart: unless-stopped
    command:
      - run
      - --server.http.listen-addr=0.0.0.0:12345
      - --storage.path=/var/lib/alloy/data
      - /etc/alloy/config.alloy
    volumes:
      - ./alloy/config.alloy:/etc/alloy/config.alloy:ro
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - alloy_data:/var/lib/alloy/data
    depends_on:
      loki:
        condition: service_healthy

  # Optional — see the note after this script for how to turn this on.
  # Not opened to the internet in Step 3's firewall rules on purpose; you
  # reach it through an SSH tunnel instead.
  grafana:
    image: grafana/grafana:11.2.0
    restart: unless-stopped
    profiles: ["observability"]
    env_file: .env
    ports:
      - "3001:3000"
    volumes:
      - grafana_data:/var/lib/grafana
      - ./grafana/provisioning:/etc/grafana/provisioning:ro
    depends_on:
      loki:
        condition: service_healthy

volumes:
  loki_data:
  grafana_data:
  alloy_data:
COMPOSE

# ── Pull images and start ────────────────────────────
docker compose -f compose.prod.yml pull
docker compose -f compose.prod.yml up -d

echo "=== Bootstrap complete ==="
```

**Loki and Alloy start automatically** — no extra step needed. They give you a
durable audit trail of logins, admin actions, and settings changes (Loki), plus
every container's raw logs in one place (Alloy) — genuinely useful for a
production deployment, and lightweight enough that there's no real reason to
skip them.

**Grafana is optional but recommended** — it's the easiest way to actually
*look at* what Loki and Alloy are collecting, with pre-built dashboards so
there's no query language to learn. It's left off by default so the server
starts leaner; turn it on once you're logged into the server (Step 12 or
later):

```bash
cd /opt/mansooba
sudo docker compose -f compose.prod.yml --profile observability up -d grafana
```

Since Grafana's port isn't opened to the internet (see Step 3), reach it from
your own computer through an SSH tunnel using the same key from Step 5:

```bash
ssh -i ~/.ssh/mansooba-key.pem -L 3001:localhost:3001 -N ec2-user@FILL_IN_YOUR_IP
```

Then open `http://localhost:3001` in your browser and log in with the
`GF_SECURITY_ADMIN_USER` / `GF_SECURITY_ADMIN_PASSWORD` you set above.

**Replace these placeholders:**

| Placeholder | Replace with |
|-------------|-------------|
| `FILL_IN_RANDOM_SECRET` | Any long random string — mash the keyboard for 40+ characters, e.g. `x7Kp2mNqR9vL4wJ8cT6yH3bF1sA5uE0d` |
| `FILL_IN_RDS_ENDPOINT` | Your RDS endpoint from Step 7 |
| `FILL_IN_DB_PASSWORD` | Your RDS master password from Step 4 |
| `FILL_IN_SES_SENDER_EMAIL` | Your verified SES sender email from Step 2.1 |
| `FILL_IN_SES_SMTP_USERNAME` | The SMTP username from Step 2.2 |
| `FILL_IN_SES_SMTP_PASSWORD` | The SMTP password from Step 2.2 |
| `FILL_IN_GRAFANA_PASSWORD` | Any password you'll remember — only matters if you turn Grafana on later |

Leave `FILL_IN_LATER` for now — you'll update the app after getting the IP in Step 10.

> `STORAGE_BUCKET` and `STORAGE_REGION` don't need replacing unless you had to pick a different bucket name in Step 6.1 due to a name collision — if so, update `STORAGE_BUCKET` to match.

Save this file as `startup-script.txt` on your Desktop (you'll copy-paste it in the next step).

---

## Step 9 — Launch your EC2 server

1. In the search bar type `EC2` → open **EC2**
2. Click **Launch instances** (the orange button)
3. Fill in:
   - Name: `mansooba-app`
4. **Application and OS Images:**
   - Under **Quick Start** the first tab should already show **Amazon Linux**
   - Select **Amazon Linux 2023 AMI** (the top result, labelled "Free tier eligible")
   - Architecture: **64-bit (x86)**
5. **Instance type:** `t2.micro` (should be pre-selected, it's free tier)
6. **Key pair:** Select `mansooba-key` (the one you created in Step 5)
7. **Network settings:** Click **Edit**
   - VPC: default
   - Subnet: any (leave as default)
   - Auto-assign public IP: **Enable**
   - Firewall: **Select existing security group** → choose `mansooba-ec2-sg`
8. **Configure storage:** leave as default (8 GB is fine)
9. Expand **Advanced details** at the bottom
10. Find **IAM instance profile** → select `mansooba-ec2-role` (the role you created in Step 6.2)
    *(Skip this and the server has no permission to talk to S3 — file attachments will fail with an "IMDS role" error even though the role exists.)*
11. Scroll down to **User data**
12. Open your `startup-script.txt` file, select all, copy → paste it into the **User data** box
13. Click **Launch instance**

> The instance starts with a **Pending** status. It becomes **Running** in about 30 seconds.

---

## Step 10 — Assign a permanent IP address

Without this step the server's IP address changes every restart.

1. In the EC2 left sidebar go to **Network & Security** → **Elastic IPs**
2. Click **Allocate Elastic IP address** → **Allocate**
3. You now have an IP (e.g. `54.123.45.67`) — **write it down**
4. With the IP selected, click **Actions** → **Associate Elastic IP address**
5. Instance: select `mansooba-app` → **Associate**

**Write this down:**
```
Server IP address:   ________________________________
```

---

## Step 11 — Update the app URL in the configuration

The startup script used `FILL_IN_LATER` for the app URL. Now that you have the real IP, update the config file on the server:

1. In the EC2 left sidebar click **Instances**
2. Select `mansooba-app` → click **Connect** (top bar)
3. Click the **EC2 Instance Connect** tab → **Connect** (opens a terminal in the browser)
4. In the terminal that opens, run:

```bash
# Replace 54.123.45.67 with your actual IP from Step 10
sudo sed -i 's|FILL_IN_LATER|http://54.123.45.67|g' /opt/mansooba/.env
sudo docker compose -f /opt/mansooba/compose.prod.yml restart backend
```

> **EC2 Instance Connect** is a browser-based SSH terminal — no software to install. It's available under the Connect button on any running instance. The default user on Amazon Linux is `ec2-user` (the Connect button handles this automatically).

---

## Step 12 — Wait for the app to start

The startup script runs in the background after launch and takes **3–5 minutes** the first time.

In the EC2 Instance Connect terminal:

```bash
sudo tail -f /var/log/user-data.log
```

Wait until you see:
```
=== Bootstrap complete ===
```

Press `Ctrl+C` to stop.

Then check that both containers are running:

```bash
sudo docker ps
```

You should see two rows — one for `mansooba-backend`, one for `mansooba-frontend`, both showing **Up**.

---

## Step 13 — Open the app

Open your browser and go to:

```
http://YOUR_SERVER_IP
```

Mansooba should load. Run through the setup wizard to create your first admin account.

**Test password-reset email:**
1. Log out → click **Forgot password**
2. Enter your verified SES email address (from Step 2.1)
3. Check your inbox — a reset email should arrive within 30 seconds

---

## Day-to-day operations

### Update to a new version

After a new Mansooba release is published:

1. Go to **EC2** → **Instances** → select `mansooba-app` → **Connect** → **EC2 Instance Connect** → **Connect**
2. Run:

```bash
cd /opt/mansooba
sudo docker compose -f compose.prod.yml pull
sudo docker compose -f compose.prod.yml up -d --remove-orphans
sudo docker image prune -f
```

### View the application logs

In EC2 Instance Connect:

```bash
sudo docker logs mansooba-backend --tail 50
sudo docker logs mansooba-frontend --tail 50
```

For security/admin events specifically (who logged in, who changed what), sign
in to the app as an admin and open **System → Logs** — that's the curated
audit trail, not raw container output. For everything else across every
container in one place, turn on Grafana (see Step 8) and open the **Container
Logs** dashboard.

### Restart the app

```bash
sudo docker compose -f /opt/mansooba/compose.prod.yml restart
```

---

## Tear down (when you're done)

Free tier lasts 12 months. When you're finished, delete everything to avoid charges:

**Delete the EC2 instance:**
1. EC2 → Instances → select `mansooba-app` → **Instance state** → **Terminate instance**

**Release the Elastic IP (you're charged for unattached IPs):**
1. EC2 → Elastic IPs → select your IP → **Actions** → **Disassociate** → then **Actions** → **Release**

**Delete the database:**
1. RDS → Databases → select `mansooba-db` → **Actions** → **Delete**
2. Uncheck "Create final snapshot" → type `delete me` to confirm → **Delete**

**Delete security groups (wait 2 minutes after the instance terminates):**
1. EC2 → Security Groups → select `mansooba-ec2-sg` → **Actions** → **Delete security groups**
2. Repeat for `mansooba-rds-sg`

**Empty and delete the attachments bucket:**
1. S3 → select `mansooba-attachments` → **Empty** → type `permanently delete` to confirm → **Empty**
2. Select `mansooba-attachments` again → **Delete** → type the bucket name to confirm → **Delete bucket**
   *(A bucket must be empty before it can be deleted — that's what the Empty step above is for.)*

---

## Troubleshooting

### The app doesn't load in the browser

The startup script might still be running. In EC2 Instance Connect, check:

```bash
sudo cat /var/log/user-data.log
```

Look for error lines or check if the script finished. If it finished but the app still doesn't load, check the security group allows port 80 (Step 3.2).

### The database shows an error

Go to RDS and confirm the database status is **Available**. Then in EC2 Instance Connect:

```bash
sudo grep DB_DSN /opt/mansooba/.env
```

Make sure the endpoint, username (`mansooba`), and password all look right.

### No password-reset email arrives

In EC2 Instance Connect:

```bash
sudo docker logs mansooba-backend | grep -i smtp
```

**"Authentication failed" error:** Your SES SMTP credentials are wrong. Re-do Step 2.2 to create new SMTP credentials, then:

```bash
sudo sed -i 's|^SMTP_USER=.*|SMTP_USER=YOUR_NEW_SMTP_USER|' /opt/mansooba/.env
sudo sed -i 's|^SMTP_PASS=.*|SMTP_PASS=YOUR_NEW_SMTP_PASS|' /opt/mansooba/.env
sudo docker compose -f /opt/mansooba/compose.prod.yml restart backend
```

**"Message rejected" or no error but no email:** SES is still in sandbox mode — only verified addresses can receive email. Verify the recipient email in SES → Verified identities (same steps as Step 2.1), or request production access (Step 2.3).

### Attaching a file to an issue fails, or the download link doesn't work

In EC2 Instance Connect:

```bash
sudo docker logs mansooba-backend | grep -i s3
sudo grep STORAGE /opt/mansooba/.env
```

Confirm `STORAGE_BUCKET` in `.env` matches the exact bucket name from Step 6.1, and that the `mansooba-s3-attachments` inline policy from Step 6.3 is still attached to `mansooba-ec2-role` (**IAM** → **Roles** → `mansooba-ec2-role` → **Permissions** tab).

**Error mentions "no EC2 IMDS role found" or "no credentials":** the instance itself has no IAM role attached — Step 9.10 was skipped, or the instance was launched before Step 6 existed. Fix it without recreating the instance: EC2 console → select `mansooba-app` → **Actions** → **Security** → **Modify IAM role** → choose `mansooba-ec2-role` → **Update IAM role**. No restart needed; retry the upload after ~30–60 seconds.

### The database never seems to stop, or logs show `"db idle auto-stop disabled"`

Check what the backend logged at startup:

```bash
sudo docker logs mansooba-backend 2>&1 | grep "db idle auto-stop"
```

If you see `"db idle auto-stop disabled"` with `dsn_host` looking correct (it should end in `.rds.amazonaws.com` and start with your DB instance identifier), the most common cause is **`RDS_INSTANCE_IDENTIFIER` accidentally set to the full RDS endpoint instead of just the instance name** — an easy mix-up since both values come from the same place (Step 7):

```
# WRONG — this is the RDS endpoint, not the identifier:
RDS_INSTANCE_IDENTIFIER=mansooba-db.cxxxxxxxx.us-east-1.rds.amazonaws.com

# RIGHT — just the instance identifier from Step 4:
RDS_INSTANCE_IDENTIFIER=mansooba-db
```

Fix it, then recreate the container (not just restart it — see the note below):
```bash
sudo sed -i 's|^RDS_INSTANCE_IDENTIFIER=.*|RDS_INSTANCE_IDENTIFIER=mansooba-db|' /opt/mansooba/.env
cd /opt/mansooba && sudo docker compose -f compose.prod.yml up -d --force-recreate backend
```

> Use `up -d --force-recreate`, not `restart` — `docker compose restart` restarts the existing container without re-reading `.env`, so an edited value silently has no effect until the container is recreated.

### RDS shows "Stopped" in the console, or the app briefly shows a "waking up" message

This is expected, not a fault — it's the auto-stop/wake-on-hit feature from Step 6.4 saving cost by stopping the database after 10 minutes of no traffic. The next request wakes it back up automatically within about a minute. If the app is stuck on "waking up" for much longer than that:

```bash
sudo docker logs mansooba-backend | grep -E 'db_auto_(stop|start)'
```

If you see a permissions error here, confirm the `mansooba-rds-lifecycle` inline policy from Step 6.4 is attached to `mansooba-ec2-role`, and that the account ID and DB instance identifier in its JSON match your actual database (**RDS** → **Databases** → `mansooba-db` → **Configuration** tab → **ARN**).

To turn this feature off entirely (e.g. so the database never stops):
```bash
echo 'RDS_AUTOSTOP_ENABLED=false' | sudo tee -a /opt/mansooba/.env
cd /opt/mansooba && sudo docker compose -f compose.prod.yml up -d --force-recreate backend
```

---

## Quick reference — what everything is

| Term | What it means |
|------|---------------|
| **EC2** | A virtual computer you rent in the cloud — Mansooba runs here |
| **S3** | Simple Storage Service — AWS's object storage; holds file attachments uploaded to issues |
| **RDS** | A managed database service — stores your data, AWS handles backups |
| **SES** | Simple Email Service — AWS's free email relay (3,000/month from EC2) |
| **Security Group** | A firewall that controls which traffic can reach a resource |
| **Key Pair** | An SSH key — lets you log into your server securely |
| **Elastic IP** | A fixed public IP address so your server URL stays the same |
| **IAM Role** | A permission identity attached to EC2 instead of using passwords |
| **User data** | A startup script EC2 runs once when the server first boots |
| **EC2 Instance Connect** | A browser-based terminal — log into your server without installing SSH |
| **SES Sandbox mode** | Default SES mode — can only send to verified email addresses |
| **Loki** | Stores the System Logs audit trail (logins, admin actions, settings changes) — starts automatically |
| **Alloy** | Collects every container's raw logs into Loki — starts automatically |
| **Grafana** | Optional dashboard for browsing what Loki and Alloy collected — off by default, recommended to turn on |

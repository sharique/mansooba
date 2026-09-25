# Post-Deployment Troubleshooting

This is a runbook for diagnosing unexpected behavior on an **already-running**
AWS deployment — the app is up, but something about it looks wrong (a
service restarting on its own, a feature that doesn't seem to be doing what
its ADR says it should, etc). It's a different scope from the
[Terraform guide's own Troubleshooting section](deploy-to-aws-terraform.md#troubleshooting)
or the [beginner guide's](deploy-to-aws-beginner.md), which cover failures
*during* `apply`/console setup — this doc starts from "deployment succeeded,
now something's odd."

Each entry follows the same shape: **Symptom → Diagnosis → Root cause → Fix
(if any)** — so a finding is fully reproducible later, not just a one-line
conclusion.

## Contents

- [RDS instance stops and starts on its own, seemingly at random](#rds-instance-stops-and-starts-on-its-own-seemingly-at-random)

---

## RDS instance stops and starts on its own, seemingly at random

**Symptom**: The RDS instance (`mansooba-db`) transitions between `stopped`
and `available` repeatedly, without an obvious pattern, and you haven't
knowingly enabled anything that would do that.

**Diagnosis — work outward from the app to AWS, not the other way round:**

1. **Is the app's own idle-autostop feature (ADR-030) even enabled?** SSH in
   and check the running container's config, not just the `.env` file on
   disk — a stale container can be running with an older `.env`:

   ```bash
   ssh -i ~/.ssh/mansooba ec2-user@<ec2_public_ip>
   cd /opt/mansooba
   grep RDS_AUTOSTOP_ENABLED .env
   docker compose -f compose.prod.yml exec backend env | grep RDS_AUTOSTOP_ENABLED
   ```

   The backend logs its decision exactly once, at boot (`cmd/server/main.go`)
   — this is the single most useful line in the whole investigation:

   ```bash
   docker compose -f compose.prod.yml logs backend | grep -i "db idle auto-stop"
   ```

   - `"msg":"db idle auto-stop enabled", ...` → the app *is* managing this
     instance's lifecycle. Read on to "app-managed" case below.
   - `"msg":"db idle auto-stop disabled","flag_enabled":false,...}` → the app
     is not touching RDS at all. Whatever is stopping/starting the instance
     is external to this codebase — go straight to step 2.
   - No line at all → the container hasn't restarted since your last `.env`
     change, or the grep window is too short; re-run with
     `docker compose -f compose.prod.yml logs --since 24h backend` or check
     `docker compose -f compose.prod.yml ps backend` for its start time.

2. **Confirm *who* is actually calling `StopDBInstance`/`StartDBInstance`**,
   regardless of what step 1 found — RDS's own event log tells you *when*,
   CloudTrail tells you *who*:

   ```bash
   aws rds describe-events --region eu-central-1 --source-type db-instance \
     --source-identifier mansooba-db --duration 20160   # last 14 days

   aws cloudtrail lookup-events --region eu-central-1 \
     --lookup-attributes AttributeKey=EventName,AttributeValue=StopDBInstance \
     --query 'Events[].CloudTrailEvent' --output text | jq '.userIdentity.arn'

   aws cloudtrail lookup-events --region eu-central-1 \
     --lookup-attributes AttributeKey=EventName,AttributeValue=StartDBInstance \
     --query 'Events[].CloudTrailEvent' --output text | jq '.userIdentity.arn'
   ```

   The app can only be the caller if the ARN contains the EC2 instance's own
   role (`mansooba-ec2-role`). Anything else — a human SSO session, a Lambda
   assuming some other role, the RDS service itself — means the app had
   nothing to do with it.

**Root cause found 2026-09-18 (this project's own account)**: `.env` had
`RDS_AUTOSTOP_ENABLED=false`. The boot log confirmed the app correctly read
that and skipped the feature entirely (`flag_enabled:false`,
`instance_configured:true` — it *would* have worked if turned on). A 90-day
CloudTrail lookback showed `mansooba-ec2-role` never once called either API.
Every `StopDBInstance` traced to
`arn:...:assumed-role/OrganizationAccountAccessRole/LambdaCrossAccountSession`
— an AWS-Organizations-level automation external to this project (this
account sits under an org with its own cost/idle guardrails; that Lambda,
not Mansooba, was stopping the database on its own schedule). Every
`StartDBInstance` traced to a human SSO session — i.e., manual restarts.
Also worth knowing so it isn't mistaken for either of the above: RDS itself
force-restarts an instance that's been stopped for 7 days straight
(`describe-events` shows this as `"...exceeding the maximum allowed time
being stopped"`) — that one's neither the app nor an external Lambda, it's
an AWS platform limit on how long `stopped` is allowed to last.

**Fix**: none needed if this is what you find and it's the behavior you
want (org-level cost control, left as-is). To have the app manage this
instance's idle lifecycle itself instead — avoiding the "someone has to
notice and manually restart it" gap:

```bash
# edit /opt/mansooba/.env: RDS_AUTOSTOP_ENABLED=true
docker compose -f compose.prod.yml up -d --force-recreate backend
docker compose -f compose.prod.yml logs backend | grep -i "db idle auto-stop"
# should now read "db idle auto-stop enabled"
```

If instead the boot log in step 1 already said **enabled** and the instance
is *still* behaving unexpectedly, that's a different, code-level bug report
— check `docs/decisions/ADR-030-db-idle-autostop.md` for the design and
compare against `internal/service/dbinstance_service.go`'s actual state
machine; two earlier live-deployment bugs in this exact feature (missing
`RDS_INSTANCE_IDENTIFIER`/`AWS_REGION` wiring, and the tracker not seeding
its boot state from a real `DescribeState` call) were found and fixed this
same way — by reading this log line first, then CloudTrail, rather than
guessing from the code alone.

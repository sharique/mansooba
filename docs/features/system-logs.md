# Feature: System Logs

## Overview

Admins have access to a System Logs page (`/system/logs`) in the System section of
the sidebar, alongside Settings and Users. It shows a durable, searchable audit trail
of security-relevant and operational events — something that previously only existed
as ephemeral process logs, if it was recorded at all.

Backed by [Grafana Loki](https://grafana.com/oss/loki/), not the application database
(ADR-031) — see the docs repo's ADR-031 for why.

## Event categories

Four categories only (spec 011's FR-011) — this is a security/operational audit
trail, not a general-purpose application error log:

| Category | Examples | Recorded from |
|---|---|---|
| `authentication` | `login_success`, `login_failed`, `refresh_rejected` | `AuthService.Login`, `AuthService.Refresh` |
| `admin_action` | `account_created`, `role_changed`, `account_enabled`, `account_disabled` | `AuthService.Register`, `AdminUserService.SetRole`/`SetActive` |
| `settings_change` | `setting_changed` (one entry per changed key) | `SettingService.Patch` |
| `db_lifecycle` | `db_auto_stop`, `db_auto_start` | `LogDBLifecycleEvent` (spec 010's idle auto-stop/wake-on-hit, ADR-030) |

Every entry captures a timestamp, category, action, outcome (`success`/`failure`),
actor, and (when applicable) a target and free-form detail. The actor is never
absent: for `db_lifecycle` entries, which have no human actor, it's the fixed value
`system`.

## Access model

Admin-only, via the same `profile.IsAdmin` gate every other System-section page and
endpoint uses (`GET /api/v1/admin/system-logs`). No separate "read-only auditor" role
exists — the closest thing to one is the optional Grafana instance described below,
which has its own separate login entirely outside this gate.

## Filtering and search

- **Filters** (combinable): category, date range, actor. The actor filter matches
  only the actor field.
- **Search** (`q`): free-text, matched against actor, target, action, and detail
  together — broader than the actor filter.

Internally, filters/search translate to a [LogQL](https://grafana.com/docs/loki/latest/query/)
query: category becomes a label selector, actor becomes a `| json | actor=~` field
filter, and `q` becomes a line filter (`|~`). User-supplied text is always passed
through `regexp.QuoteMeta` so it behaves as a literal substring match.

## Optional: Grafana for deep log exploration

The in-app System Logs page is deliberately narrow — category, date range, actor,
and keyword filters only (see [Filtering and search](#filtering-and-search) above).
That's enough for "did this admin action happen, and who did it," but not for
open-ended investigation: correlating events across categories, ad-hoc LogQL
(regex, aggregations, rate-over-time), or a real-time tail.

For that, an optional [Grafana](https://grafana.com/oss/grafana/) instance is
available, wired to read the same Loki store System Logs already writes to — no
separate ingestion, no second copy of the data. It's aimed at two audiences beyond
what the in-app page serves:

- **System admins** who need to go past the built-in filters for a specific
  investigation (e.g. "show me every `failure` outcome across all categories in
  the last 24 hours, grouped by actor").
- **The infra team**, who may not have (or want) an in-app admin account at all —
  Grafana has its own separate login, entirely outside the application's
  `profile.IsAdmin` gate, so infra access doesn't require provisioning an
  application user.

### Enabling it

Not started by default — it's a genuinely optional add-on, gated behind a Docker
Compose [profile](https://docs.docker.com/compose/how-tos/profiles/):

```bash
# Local dev (compose.yml)
docker compose --profile observability up -d grafana

# Production (compose.prod.yml, on the deployed instance)
docker compose -f compose.prod.yml --profile observability up -d grafana
```

Loki is pre-provisioned as Grafana's default datasource
(`grafana/provisioning/datasources/loki.yaml`), and a **"System Logs" dashboard**
is pre-provisioned too (`grafana/provisioning/dashboards/`) — it's the first thing
you land on, not something you have to build. It has two panels: a log stream
(`{app="mansooba"}`, last 24h, newest first) and an entries-by-category graph. No
LogQL knowledge is required just to look at what's there.

Loki itself has no "show everything" default the way a normal dashboard homepage
does — every query needs an explicit stream selector — so **Explore** (for queries
beyond what the dashboard shows) always needs a query typed in, even an empty one
won't return anything on its own. Example queries to build on:

```logql
{app="mansooba"}                                    # everything
{app="mansooba"} | json | outcome="failure"          # every failed action, any category
{app="mansooba",category="authentication"} | json    # parsed fields (actor, action, ...)
```

### Container logs (raw stdout/stderr, via Alloy)

System Logs and the Grafana dashboards above only ever contain what this
application deliberately records — the four categories in
[Event categories](#event-categories). They say nothing about a container that
crashed on startup, panicked, or is spewing framework noise; that's a different
kind of signal, living in each container's own stdout/stderr.

[Grafana Alloy](https://grafana.com/docs/alloy/latest/) (`alloy/config.alloy`)
tails every container's logs via the Docker Engine API and ships them into the
*same* Loki instance — but under a different label, `job="dockerlogs"`, never
`app="mansooba"`. That separation is deliberate: System Logs' own queries
(`loki_systemlog_repository.go`'s `buildLogQL`) assume every line under
`app="mansooba"` is the structured JSON `SystemLogEntry` shape and parse it with
`| json` — mixing in raw, unstructured container output under that same label
would break that parsing and pollute the audit trail. Same Loki, same Grafana,
cleanly separate streams.

Unlike Grafana, Alloy runs **by default** (not gated behind the `observability`
profile) — a log collector only earns its keep if it's actually running when
something crashes, not just when someone happens to have Grafana open
afterward. A pre-provisioned **"Container Logs"** dashboard (with a
`compose_service` filter — `backend`, `frontend`, `loki`, `alloy`, ...) shows the
raw stream; enabling Grafana is still what you need to actually look at it.

Alloy needs read access to the Docker socket to discover containers and tail
their logs — the same requirement any Docker-log-shipping tool has (Promtail,
Fluentd, ...). Docker sockets don't support partial read-only enforcement at the
bind-mount level, so this is a real privilege-scope trade-off: whoever can reach
that socket can, in principle, do anything the Docker API allows. Acceptable for
a single-operator deployment; a more adversarial/multi-tenant environment would
warrant a docker-socket-proxy in front of it instead.

### Access

| Environment | URL | Credentials |
|---|---|---|
| Local dev | http://localhost:3001 | `admin` / `admin` (default — local dev only) |
| Production | *(no public URL — see below)* | `GF_SECURITY_ADMIN_USER` / `GF_SECURITY_ADMIN_PASSWORD` from `.env` |

In production, Grafana's host port is **deliberately not opened** in the EC2
security group (`terraform/modules/security`) — an audit-log viewer shouldn't be
reachable from the public internet by default, the same reasoning that already
keeps Loki itself off the public security group. Reach it via an SSH tunnel
instead:

```bash
ssh -L 3001:localhost:3001 ec2-user@<instance-ip>
# then open http://localhost:3001 in a local browser
```

The admin password is fetched from SSM (`/mansooba/GRAFANA_ADMIN_PASSWORD`) at
instance boot if the operator has set it there, otherwise `user-data.sh` generates
a fresh random one on every boot — set the SSM parameter for a password that
survives instance restarts.

### What it does *not* change

Grafana is read-only against Loki and has no write path into System Logs, the
application, or its database — enabling it cannot affect `Record()`, retention, or
anything else this feature already does. Alloy only *adds* a second stream
(`job="dockerlogs"`) via its own independent push path; it never reads from or
writes to System Logs' own `app="mansooba"` stream. Both are purely additional
query surfaces over data that already exists (or, for Alloy, data Docker was
already generating regardless).

## Retention

Admin-configurable via the existing Settings page (`system_log_retention_days`,
default 90 days, minimum 1, no fixed maximum) — the same `global_settings`
key-value mechanism every other platform setting uses. A background goroutine
(`startSystemLogRetentionSync`, mirroring `startRevokedTokenCleanup`'s shape)
reconciles Loki's runtime-overrides file with the current setting; Loki's own
compactor performs the actual expiry. A reduced retention period taking effect
against already-stored entries is subject to Loki's compaction schedule, not
instantaneous.

This retention period is a **per-tenant** Loki setting, and this deployment has
exactly one tenant (`auth_enabled: false` means every write lands in the same
implicit tenant, `fake`) — so it applies to *every* stream in Loki, not just
`app="mansooba"`. Lowering `system_log_retention_days` for audit-log-storage
reasons also shortens how long Alloy's `job="dockerlogs"` container logs stick
around, even though they aren't System Log entries. There's currently no way to
give the two streams independent retention without a second Loki tenant.

## Failure isolation

Recording a System Log entry is best-effort: a failure to write one — for example, if
Loki is unreachable — never blocks, delays, or fails the action it's describing
(login, admin action, settings change, or database lifecycle transition). The write
itself happens in a fire-and-forget goroutine (`SystemLogService.Record`). A write
failure is independently observable via the application's existing structured
(`zap`) logging, even though it never reaches System Logs itself.

## Health check

Since Loki is this project's first new long-running service (previously, every
feature reused existing infrastructure — see ADR-030's and ADR-017's precedent of
avoiding new deployable units), the existing `/health` endpoint was extended
(`HealthHandler.WithLoki`) to also report Loki reachability. Unlike the database
check, a Loki outage is reported (`loki: "error"`) but does **not** flip the overall
health status to `503` — the application keeps operating normally without it,
consistent with the best-effort design above.

## What's explicitly out of scope

- General application error/exception logging (this stays a security/operational
  audit trail only)
- Export (e.g. CSV download) of log entries
- A live-streaming/real-time tail — this is a paginated, on-demand list like the
  existing per-issue activity feed
- Backfilling historical events from before this feature shipped

## API endpoint

See [api.md](../arch/api.md). Key route:

- `GET /api/v1/admin/system-logs` — query params: `category`, `from`, `to`, `actor`,
  `q`, `page`, `size`. Returns `{entries, total, page, size}`, matching the shape of
  the existing `GET /admin/users` endpoint.

## Related decisions

(In the docs repo, `sharique/mansooba-docs` — this is a separate git tree from
`sharique/mansooba`, so these aren't relative links.)

- ADR-031 — why Loki, not the application database
- ADR-030 — the database auto-stop/wake-on-hit feature whose events this
  feature now durably captures

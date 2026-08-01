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
exists.

## Filtering and search

- **Filters** (combinable): category, date range, actor. The actor filter matches
  only the actor field.
- **Search** (`q`): free-text, matched against actor, target, action, and detail
  together — broader than the actor filter.

Internally, filters/search translate to a [LogQL](https://grafana.com/docs/loki/latest/query/)
query: category becomes a label selector, actor becomes a `| json | actor=~` field
filter, and `q` becomes a line filter (`|~`). User-supplied text is always passed
through `regexp.QuoteMeta` so it behaves as a literal substring match.

## Retention

Admin-configurable via the existing Settings page (`system_log_retention_days`,
default 90 days, minimum 1, no fixed maximum) — the same `global_settings`
key-value mechanism every other platform setting uses. A background goroutine
(`startSystemLogRetentionSync`, mirroring `startRevokedTokenCleanup`'s shape)
reconciles Loki's runtime-overrides file with the current setting; Loki's own
compactor performs the actual expiry. A reduced retention period taking effect
against already-stored entries is subject to Loki's compaction schedule, not
instantaneous.

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

# Feature: Issues

## Overview

Issues are the core work item in Mansooba. Each issue belongs to a project and can
represent a task, story, bug, or epic. Issues support rich metadata (type, status,
priority, story points, assignee), free-text search, label tagging, directional
relationship links between issues, and file attachments.

## Implementation details

### Issue relations

Four relation types: `blocks`, `is_blocked_by`, `relates_to`, `duplicates`.

- Reciprocal links are maintained automatically: creating a `blocks` link from A→B also
  creates an `is_blocked_by` link from B→A
- When an issue is deleted, all its relation records (both directions) are cascade-deleted

### Labels

- Labels are project-scoped — a label created in Project A is not visible in Project B
- Issues can have multiple labels; labels can be used to filter the issue list

### Search

- Text search runs across issue title and description
- Can be combined with type, status, priority, and label filters simultaneously

### Status workflow

```
backlog → todo → in_progress → in_review → done
```

Status transitions are unconstrained — any status can be set directly from any other.

### Story points

- Integer field; used by sprint burndown and velocity charts
- No range constraint is enforced

### Attachments

- Files are uploaded to an S3-compatible bucket, never the app server's local disk —
  real AWS S3 in production, LocalStack in local dev, through the same code path
  (`STORAGE_ENDPOINT` unset = real S3; set = LocalStack)
- Object keys follow `{projectKey}/{issueKey}/{uuid}.{ext}` — the original filename is
  never used as the storage key, so two attachments with identical names on the same
  issue never collide; the original filename is preserved separately for display and
  download
- Upload is backend-mediated (not a direct-to-S3 presigned PUT) so the server can
  validate content by magic bytes, not just the declared MIME type or file extension —
  a mismatch (e.g. a `.png` that isn't actually a PNG) is rejected
- Limits: 10 MB per file, 20 attachments per issue (both server-side, not configurable
  from the UI)
- Download returns a short-lived presigned S3 URL as JSON, not a redirect — a redirect
  can't carry this app's Bearer auth header, and a script-initiated `fetch` can't read a
  cross-origin redirect's `Location` either. Project membership is checked *before* the
  URL is generated, so a rejected caller never receives a working link
- Deleting an issue cascade-deletes both the attachment rows and their S3 objects (a
  batched `S3.DeleteObjects` call) — never just the DB rows, which would orphan objects
  in the bucket
- Delete permission: the uploader, or a project admin (enforced server-side); the UI
  currently only surfaces the delete control to the uploader, matching the same known
  gap as comments (no per-project role surfaced to the frontend yet beyond the global
  `IsAdmin` superadmin flag)

## API endpoints

See [api.md](../arch/api.md). Key issue routes are under
`/api/v1/projects/:id/issues/`. Attachment routes are under
`/api/v1/issues/:id/attachments/`.

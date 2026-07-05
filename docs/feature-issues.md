# Feature: Issues

## Overview

Issues are the core work item in Mansooba. Each issue belongs to a project and can
represent a task, story, bug, or epic. Issues support rich metadata (type, status,
priority, story points, assignee), free-text search, label tagging, and directional
relationship links between issues.

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

## API endpoints

See [arch-api.md](arch-api.md). Key issue routes are under
`/api/v1/projects/:id/issues/`.

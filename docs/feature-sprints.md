# Feature: Sprints, Backlog & Board

## Overview

Sprints are time-boxed work periods within a project. Each sprint has a name, start
date, end date, and a lifecycle status. The kanban board shows all issues in the active
sprint grouped by status column. The backlog holds all issues not yet assigned to a
sprint.

## Implementation details

### Sprint lifecycle

```
planning → active → completed
```

- A project can have at most one sprint in `active` state at a time — the backend
  enforces this constraint and returns 409 if a second sprint is activated while one is
  already active

### Sprint completion

When a sprint is completed, unfinished issues (status not `done`) are automatically
migrated:
- To the next sprint, if one is specified in the completion request
- Back to the backlog, if no next sprint is specified

### Burndown chart

- Data: story points remaining over time within the sprint
- Computed from the issue activity log — each event that moves an issue to `done` is
  recorded with a timestamp and subtracted from the remaining total

### Velocity chart

- Data: committed story points vs completed story points per sprint
- Committed = total story points of all issues in the sprint at start
- Completed = story points of issues that reached `done` status by the end of the sprint

## API endpoints

See [arch-api.md](arch-api.md). Key sprint routes are under
`/api/v1/projects/:id/sprints/`.

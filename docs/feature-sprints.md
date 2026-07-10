# Feature: Sprints, Backlog & Board

## Overview

Sprints are time-boxed work periods within a project. Each sprint has a name, an
optional goal, start date, end date, and a lifecycle status. The kanban board shows all
issues in the active sprint grouped by status column. The backlog holds all issues not
yet assigned to a sprint.

## Implementation details

### Sprint lifecycle

```
planning → active → completed
```

- A project can have at most one sprint in `active` state at a time — the backend
  enforces this constraint and returns 409 if a second sprint is activated while one is
  already active

### Sprint creation validation

- Name, start date, and end date are all required, dates must be today or later, and the
  end date must be strictly after the start date
- **These rules are enforced in the frontend only** (`SprintForm.vue`) — the backend's
  `CreateSprintRequest` only validates `Name` (required); `StartDate`/`EndDate` have no
  server-side validation at all. A direct API call can create a sprint with a past date
  range or an end date before the start date.

### Sprint completion

When a sprint is completed, unfinished issues (status not `done`) are automatically
migrated:
- To the next sprint, if one is specified in the completion request
- Back to the backlog, if no next sprint is specified

This means a genuinely completed sprint (via the normal `Complete` flow) ends up with
only `done` issues still attached to it — which affects how "committed" reads on the
velocity chart below.

### Burndown chart

- Data: story points remaining per day, from sprint start to end
- Computed by walking each day in the sprint's date range and checking each issue's
  `CompletedAt` timestamp (set once, when the issue's status transitions to `done`)
  against that day — not from a stream of activity-log events

### Velocity chart

- Data: committed story points vs completed story points per sprint
- Committed = sum of story points of whatever issues are *currently* attached to the
  sprint (not a snapshot taken "at sprint start" — there's no such snapshot stored).
  Because `Complete()` migrates unfinished issues out of the sprint, for a sprint that
  went through the normal completion flow this ends up equal to Completed in practice.
- Completed = story points of issues with status `done`

## API endpoints

See [arch-api.md](arch-api.md). Most sprint CRUD/lifecycle routes (get, update, delete,
start, complete, list issues, burndown) are under
`/api/v1/projects/:key/sprints/...`. Backlog (`GET /api/v1/projects/:key/backlog`) and
velocity (`GET /api/v1/projects/:key/velocity`) are sibling routes directly under
`/api/v1/projects/:key/`, not nested under `/sprints/`.

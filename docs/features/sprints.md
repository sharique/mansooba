# Feature: Sprints, Backlog & Board

## Overview

Sprints are time-boxed work periods within a project. Each sprint has a name, an
optional goal, start date, end date, and a lifecycle status. The sprint board shows the
project's tasks grouped by status column (tasks with the Backlog status are not on it). The backlog holds all issues not
yet assigned to a sprint.

## Implementation details

### Sprint lifecycle

```
planning → active → completed
```

- A project can have at most one sprint in `active` state at a time — the backend
  enforces this constraint and returns 409 if a second sprint is activated while one is
  already active

### Board

- Four columns: To Do, In Progress, In Review, Done. Backlog tasks are not on the board.
  Inside a column, cards are ordered by priority (critical first) and then by issue number;
  the server sorts them and the browser applies the same rule to a card that has just moved
  (both are tested against `testdata/board-order-cases.json`).
- **Dragging a card to another column changes its status.** The card moves at once and is
  saved with the normal task update (`PUT /projects/:key/issues/:id` with only `status`), so
  the completion time and the activity history behave exactly as when the status is edited on
  the task page. If the save fails the card goes back and an error is shown. Pressing
  Escape, or releasing outside a column, cancels the drag. Clicking a card still opens it.
- **Keyboard and screen readers:** every card has a "Move to…" button that lists the other
  columns (Tab, Enter or Space to choose, Escape to close). Each move, and each failed move,
  is announced in a live region.
- **Cards of completed sprints are locked.** Finishing a sprint leaves its done tasks in it,
  and moving one out of Done would change that sprint's velocity and burndown, so the board
  shows them with a lock ("Sprint completed") and does not let them be moved. This is a
  board rule only: the task page can still edit them, and a board that was loaded before a
  sprint was completed still looks movable until it is refreshed.
- The board never updates live. It refreshes itself quietly when you come back to the tab or
  window after at least 30 seconds, unless a move is still being saved.
- On a touch screen, press and hold a card briefly, then drag. To cancel, drop it back on its
  own column.

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

- Shown on the project's Reports page, not the board. A sprint dropdown defaults to the sprint
  in progress (marked "current"), or "None" when no sprint is active; completed sprints can
  be picked too. Planned sprints are not listed because they have no burndown yet.
- A sprint with no days to show (for example one starting in the future) returns an empty
  `data` list, never `null`
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

See [api.md](../arch/api.md). Most sprint CRUD/lifecycle routes (get, update, delete,
start, complete, list issues, burndown) are under
`/api/v1/projects/:key/sprints/...`. Backlog (`GET /api/v1/projects/:key/backlog`) and
velocity (`GET /api/v1/projects/:key/velocity`) are sibling routes directly under
`/api/v1/projects/:key/`, not nested under `/sprints/`.

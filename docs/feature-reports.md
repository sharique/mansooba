# Feature: Reports

## Overview

The `/reports` page provides cross-sprint velocity analysis for any project you have
access to. Select a project from the dropdown to view a bar chart of committed vs
completed story points across all completed sprints.

## Implementation details

### Velocity chart data source

- Data comes from all `completed` sprints for the selected project
- Each bar pair: committed story points (total at sprint start) and completed story
  points (total of `done` issues at sprint end)

### Chart implementation

The chart is implemented in pure CSS — no external charting library is used. This keeps
the bundle size minimal and avoids a third-party dependency for a single chart type.

- Bar heights are set as inline CSS custom properties (`--height: X%`) on each bar
  element
- The bar group is a flex container; bars scale proportionally to the maximum value
  across all displayed sprints

## API endpoints

See [arch-api.md](arch-api.md). Report data is derived from sprint endpoints under
`/api/v1/projects/:id/sprints/`.

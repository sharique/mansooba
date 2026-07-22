# Feature: Reports

## Overview

The `/reports` page provides cross-sprint velocity analysis for any project you have
access to. Select a project from the dropdown to view a bar chart of committed vs
completed story points across all completed sprints. The same chart is also available
without a dropdown at `/projects/:key/reports`, scoped to that project via the route.

## Implementation details

### Velocity chart data source

- Data comes from all `completed` sprints for the selected project
- Each bar pair: committed story points and completed (`done`-status) story points. Note
  "committed" is not a snapshot taken at sprint start — see sprints.md's Velocity
  chart section for why, in practice, committed tends to equal completed for sprints that
  went through the normal completion flow.

### Chart implementation

The chart is implemented in pure CSS — no external charting library is used. This keeps
the bundle size minimal and avoids a third-party dependency for a single chart type.

- Bar heights are set as an inline `height` style in pixels, computed as
  `(value / maxValue) * CHART_HEIGHT_PX` — not a CSS custom property or a percentage
- The bar group is a flex container; bars scale proportionally to the maximum value
  across all displayed sprints

## API endpoints

See [api.md](../arch/api.md). Report data comes from `GET
/api/v1/projects/:key/velocity` — a sibling route under `/projects/:key/`, not nested
under `/sprints/`.

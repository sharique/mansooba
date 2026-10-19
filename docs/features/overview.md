# Feature: Project Overview

## Overview

The project's main page (`/projects/:key`) is an at-a-glance summary with three sections that load and fail independently: the active sprint's progress, the sprint's open tasks, and recent activity. The member list and the project name/description editor live on the Members page (`/projects/:key/members`).

Only members of the project can see it; everyone else gets 403 (404 for an unknown key). Administrators get no extra read access.

## What it shows

### Current sprint

- The active sprint's name, start and end dates, and an **Overdue** badge from the day after the end date (UTC calendar days, the same rule as the Sprints page)
- Tasks and story points in the board's four columns: To Do, In Progress, In Review, Done. Tasks without points count as 0 points; a task with Backlog status that sits in a sprint counts as To Do
- With no active sprint the section says so and links to the Sprints page

### Open tasks

- Up to 10 tasks of the **active sprint** that are not Done, ordered by priority (critical, high, medium, low), then status (In Review, In Progress, To Do), then most recently updated
- Each row shows key, title, status, priority and assignee and opens the task. A **Goto board** link is shown whenever a sprint is active

### Recent activity

- Up to 10 events from the last 5 days, newest first, with two tabs: **All activity** (every task in the project) and **Current sprint** (tasks currently in the active sprint)
- Two kinds of event qualify: a task moving into In Progress, In Review or Done (a reopened task, for example Done to In Review, counts), and a comment being added. A comment entry never shows the comment's text. Other changes (assignee, priority, labels, moves into To Do or Backlog) are not listed
- Switching tabs reloads only this section

## Implementation details

- Three endpoints, one per section: `GET /projects/:key/overview/sprint`, `/tasks` and `/activity?scope=all|sprint` (see the [API reference](../arch/api.md#project-overview)). Each section therefore has its own loading, empty and error state and a Retry button
- Counts and ordering are computed in the database: one grouped query for the status counts, `CASE` expressions for the open-task ordering, and a join of the activity events to the project's issues for the activity list. The queries are portable SQL and are tested on SQLite and PostgreSQL
- Recent activity reads the activity events the app already records for every task (`status_changed`, `comment_added`); nothing new is stored. One composite index on `(issue_id, created_at)` keeps the lookup fast
- The "Current sprint" tab uses the task's sprint at the time you open the page, not the sprint it was in when the event happened
- The page does not update live; reloading shows the latest

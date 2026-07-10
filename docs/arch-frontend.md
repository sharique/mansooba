# Frontend Architecture

See [arch-overview.md](arch-overview.md) for the project-level overview.

## Build mode

The frontend is a **Single-Page Application** (SPA). SSR is disabled (`ssr: false` in `nuxt.config.ts`). `npm run generate` produces a static HTML/JS/CSS bundle that nginx serves from `/usr/share/nginx/html`. The nginx config proxies all `/api/*` requests to the backend container, so the browser only ever talks to one origin.

---

## Layers

### `services/`

Thin wrappers around `$api` (a `$fetch` proxy defined in `plugins/api.ts`). Each file mirrors one backend resource:

| Service file | Backend resource |
|-------------|-----------------|
| `auth.service.ts` | `/auth/*` |
| `issues.service.ts` | `/projects/:key/issues` |
| `projects.service.ts` | `/projects` |
| `sprints.service.ts` | `/projects/:key/sprints` |
| `comments.service.ts` | `/issues/:id/comments` |
| `activity.service.ts` | `/issues/:id/activity`, `/auth/me/activity` |
| `labels.service.ts` | `/projects/:key/labels`, `/issues/:id/labels` |
| `notifications.service.ts` | `/notifications` |
| `attachments.service.ts` | `/issues/:id/attachments` |
| `relations.service.ts` | `/issues/:id/relations` |
| `settings.service.ts` | `/settings` |
| `setup.service.ts` | `/setup/*` |
| `board.service.ts` | `/projects/:key/board` |
| `backlog.service.ts` | `/projects/:key/backlog` |
| `reports.service.ts` | `/projects/:key/velocity`, `/projects/:key/sprints/:id/burndown` |

### `stores/`

Pinia stores hold fetched state and expose actions that call services. They are the single source of truth for their domain.

| Store | State held |
|-------|-----------|
| `auth` | Current user profile, login/logout actions |
| `issues` | Issue list and active issue for a project |
| `projects` | Project list, active project, member list |
| `sprints` | Sprints, active sprint, burndown/velocity data |
| `comments` | Comments for the currently viewed issue |
| `labels` | Labels for a project and for individual issues |
| `notifications` | Unread notification list |
| `attachments` | Attachments for the currently viewed issue |
| `relations` | Task relations for the currently viewed issue |
| `settings` | Org-wide global settings; `getByKey(key)` computed getter |
| `theme` | Current theme (`light` / `dark`) |

### `pages/`

File-based Nuxt routes using Composition API. Pages trigger store actions and compose components.

| Route | Page file | Description |
|-------|-----------|-------------|
| `/` | `pages/index.vue` | Dashboard — My Desk (assigned issues, recent activity) |
| `/login` | `pages/login.vue` | Login form |
| `/register` | `pages/register.vue` | Registration form |
| `/setup` | `pages/setup.vue` | First-run superadmin wizard |
| `/settings` | `pages/settings.vue` | Profile, appearance, and organisation tabs |
| `/reports` | `pages/reports.vue` | Cross-project reports |
| `/projects/:key` | `pages/projects/[key]/index.vue` | Project overview |
| `/projects/:key/board` | `pages/projects/[key]/board.vue` | Kanban board |
| `/projects/:key/backlog` | `pages/projects/[key]/backlog.vue` | Sprint backlog |
| `/projects/:key/reports` | `pages/projects/[key]/reports.vue` | Velocity + burndown charts |
| `/projects/:key/settings` | `pages/projects/[key]/settings.vue` | Project settings |
| `/projects/:key/issues/:id` | `pages/projects/[key]/[id].vue` | Issue detail (comments, attachments, relations, activity) |

### `components/`

Vue SFCs grouped by domain. Nuxt auto-imports them with a `<Directory><ComponentName>` naming convention — for example, `components/relations/RelationsPanel.vue` → `<RelationsRelationsPanel>`.

| Directory | Contents |
|-----------|---------|
| `auth/` | Login/register forms |
| `backlog/` | BacklogList, BacklogIssueCard, sprint assignment |
| `board/` | KanbanBoard, KanbanColumn, KanbanCard |
| `charts/` | BurndownChart, VelocityChart (Chart.js wrappers) |
| `common/` | EmptyState, UserAvatar (with OKLCH initials fallback) |
| `dashboard/` | DashboardGrid, MyIssuesWidget, RecentActivityWidget |
| `issues/` | IssueForm, IssueDetail, AttachmentSection, CommentSection/CommentItem, ActivityFeed, RelatedTasksSection, IssueLabelPicker |
| `labels/` | LabelPicker, LabelBadge, LabelList |
| `layout/` | Sidebar, TopBar, ThemeToggle, NotificationBell |
| `projects/` | ProjectForm, ProjectMemberList |
| `relations/` | RelationItem, RelationsPanel (link/unlink tasks) |
| `reports/` | report-specific charts and stat cards |
| `sprints/` | SprintList, SprintCard, SprintForm |
| `ui/` | Generic primitives (buttons, modals, inputs, toasts) |

### `composables/`

| Composable | Purpose |
|-----------|---------|
| `useAvatarColor` | Derives an OKLCH colour from a user name for initials avatars |
| `useMarkdown` | Renders comment body markdown |
| `useTheme` | Reads/toggles the current theme |
| `useTimeFormatter` | Relative time formatting (e.g. "2 hours ago") |
| `useToast` | Push toast notifications from anywhere in the app |

### `middleware/`

Two global middleware files run on every navigation:

- **`auth.global.ts`** — redirects unauthenticated users to `/login`. Skips `/login`, `/register`, `/setup`.
- **`setup.global.ts`** — calls `GET /api/v1/setup/status` on first page load; redirects to `/setup` if no users exist yet.

### `plugins/`

| Plugin | Purpose |
|--------|---------|
| `api.ts` | Creates the `$api` composable — a `$fetch` instance pre-configured with the API base URL and auth header injection |
| `auth.init.client.ts` | Restores auth state from localStorage on app boot |
| `theme.client.ts` | Applies the saved theme class to `<html>` before first render |

### `types/`

| File | Contents |
|------|---------|
| `domain.types.ts` | All backend entity shapes: `Issue`, `Project`, `User`, `Sprint`, `Comment`, `Label`, `Attachment`, `TaskRelation`, `GlobalSetting`, etc. |
| `api.types.ts` | Pagination wrappers, list-query param types |
| `auth.types.ts` | `UserProfileResponse`, `LoginResponse`, `TokenPair` |
| `setup.types.ts` | `SetupStatusResponse`, `CreateSuperAdminRequest` |

---

## Design system

The UI uses an OKLCH colour palette for WCAG AA-compliant contrast at all theme levels. Design tokens are defined in `assets/css/` and consumed via CSS custom properties throughout components.

The `UserAvatar` component renders a profile photo when `avatar_url` is set, or falls back to coloured initials derived by `useAvatarColor` — a deterministic hash of the user's name to a perceptually-uniform OKLCH hue.

export interface User {
  id: number
  name: string
  email: string
}

export interface Project {
  id: number
  key: string
  name: string
  description: string
  ownerId: number
}

export interface ProjectMember {
  id: number
  projectId: number
  userId: number
  role: string
  user?: User
}

// Shape returned by GET /projects/:key/members
export interface MemberResponse {
  user_id: number
  name: string
  email: string
  role: string
}

export interface Issue {
  id: number
  key: string
  project_id: number
  title: string
  description: string
  type: 'task' | 'story' | 'bug' | 'epic'
  status: 'todo' | 'in_progress' | 'in_review' | 'done' | 'backlog'
  priority: 'low' | 'medium' | 'high' | 'critical'
  assignee_id?: number
  reporter_id: number
  story_points?: number
  sprint_id?: number | null
  created_at: string
  completed_at?: string | null
}

// ── Domain constant objects ───────────────────────────────────────────────────
// Named-value maps for use in scripts and templates.
// The union types above remain the canonical source for TypeScript narrowing.

export const IssueType = {
  Task:  'task',
  Story: 'story',
  Bug:   'bug',
  Epic:  'epic',
} as const

export const IssuePriority = {
  Low:      'low',
  Medium:   'medium',
  High:     'high',
  Critical: 'critical',
} as const

export const IssueStatus = {
  Backlog:    'backlog',
  Todo:       'todo',
  InProgress: 'in_progress',
  InReview:   'in_review',
  Done:       'done',
} as const

export const SprintStatus = {
  Planning:  'planning',
  Active:    'active',
  Completed: 'completed',
} as const

// ── Sprint ────────────────────────────────────────────────────────────────────

export type SprintStatus = 'planning' | 'active' | 'completed'

export interface Sprint {
  id: string
  project_id: string
  name: string
  goal: string
  status: SprintStatus
  start_date: string | null
  end_date: string | null
  created_at: string
  updated_at: string
  issue_count?: number
  total_story_points?: number
}

// ── Burndown ─────────────────────────────────────────────────────────────────

export interface BurndownPoint {
  date: string
  remaining_points: number
}

export interface BurndownData {
  sprint_id: string
  sprint_name: string
  start_date: string
  end_date: string
  total_points: number
  data: BurndownPoint[]
}

// ── Notifications ─────────────────────────────────────────────────────────────

export interface Notification {
  id: number
  recipient_id: number
  actor_id: number
  issue_id: number
  issue_key: string
  project_key: string
  comment_id: number
  read: boolean
  created_at: string
}

// ── Comments ──────────────────────────────────────────────────────────────────

export interface Comment {
  id: number
  issue_id: number
  author_id: number
  author_name: string | null
  body: string
  created_at: string
  updated_at: string
}

// ── Activity ──────────────────────────────────────────────────────────────────

export interface ActivityEvent {
  id: number
  issue_id: number
  actor_id: number
  actor_name: string
  issue_key: string
  issue_title: string
  kind: string
  old_value?: string
  new_value?: string
  created_at: string
}

export const ActivityKind = {
  StatusChanged:      'status_changed',
  AssigneeChanged:    'assignee_changed',
  PriorityChanged:    'priority_changed',
  SprintChanged:      'sprint_changed',
  StoryPointsChanged: 'story_points_changed',
  CommentAdded:       'comment_added',
  LabelAdded:         'label_added',
  LabelRemoved:       'label_removed',
} as const

// ── Labels ────────────────────────────────────────────────────────────────────

export interface Label {
  id: number
  project_id: number
  name: string
  color: string
  created_at: string
}

export const LabelPalette = [
  '#e11d48', '#f97316', '#eab308', '#22c55e',
  '#06b6d4', '#3b82f6', '#8b5cf6', '#ec4899',
  '#6b7280', '#78716c', '#0f172a', '#ffffff',
] as const

// ── User Profile ──────────────────────────────────────────────────────────────

export interface UserProfileResponse {
  id: number
  name: string
  email: string
  avatar_url: string
  timezone: string
  created_at: string
}

export interface UpdateProfilePatch {
  full_name?: string
  avatar_url?: string
  timezone?: string
}

// ── Search / Filters ──────────────────────────────────────────────────────────

export interface IssueFilters {
  q?: string
  type?: string
  status?: string
  priority?: string
  assignee_id?: number
  label_id?: number
}

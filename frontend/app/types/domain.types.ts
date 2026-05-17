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
  projectId: number
  title: string
  description: string
  type: 'task' | 'story' | 'bug' | 'epic'
  status: 'todo' | 'in_progress' | 'in_review' | 'done' | 'backlog'
  priority: 'low' | 'medium' | 'high' | 'critical'
  assigneeId?: number
  reporterId: number
  storyPoints?: number
  sprint_id?: number
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

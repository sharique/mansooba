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
}

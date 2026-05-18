import type { Issue } from '~/types/domain.types'

export interface CreateIssueRequest {
  title: string
  description?: string
  type: 'task' | 'story' | 'bug' | 'epic'
  priority: 'low' | 'medium' | 'high' | 'critical'
  status?: string
  assignee_id?: number
  story_points?: number
}

export interface UpdateIssuePayload extends Partial<CreateIssueRequest> {
  sprint_id?: number | null
}

export interface IssueListQuery {
  status?: string
  type?: string
  priority?: string
  assignee_id?: number
}

export const issuesService = {
  list(key: string, filters?: IssueListQuery): Promise<Issue[]> {
    const { $api } = useNuxtApp()
    return $api<Issue[]>(`/projects/${key}/issues`, { query: filters })
  },

  get(key: string, id: number): Promise<Issue> {
    const { $api } = useNuxtApp()
    return $api<Issue>(`/projects/${key}/issues/${id}`)
  },

  create(key: string, data: CreateIssueRequest): Promise<Issue> {
    const { $api } = useNuxtApp()
    return $api<Issue>(`/projects/${key}/issues`, { method: 'POST', body: data })
  },

  update(key: string, id: number, data: UpdateIssuePayload): Promise<Issue> {
    const { $api } = useNuxtApp()
    return $api<Issue>(`/projects/${key}/issues/${id}`, { method: 'PUT', body: data })
  },

  remove(key: string, id: number): Promise<void> {
    const { $api } = useNuxtApp()
    return $api(`/projects/${key}/issues/${id}`, { method: 'DELETE' })
  },
}

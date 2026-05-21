import type { Issue, IssueFilters } from '~/types/domain.types'

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

  search(projectKey: string, filters: IssueFilters): Promise<Issue[]> {
    const { $api } = useNuxtApp()
    const params = new URLSearchParams()
    if (filters.q)           params.set('q', filters.q)
    if (filters.type)        params.set('type', filters.type)
    if (filters.status)      params.set('status', filters.status)
    if (filters.priority)    params.set('priority', filters.priority)
    if (filters.assignee_id) params.set('assignee_id', String(filters.assignee_id))
    if (filters.label_id)    params.set('label_id', String(filters.label_id))
    const qs = params.toString()
    return $api<Issue[]>(`/projects/${projectKey}/issues${qs ? '?' + qs : ''}`)
  },
}

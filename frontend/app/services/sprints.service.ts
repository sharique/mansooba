import type { Sprint, BurndownData } from '~/types/domain.types'

export interface CreateSprintPayload {
  name: string
  goal?: string
  start_date?: string
  end_date?: string
}

export interface UpdateSprintPayload {
  name?: string
  goal?: string
  start_date?: string
  end_date?: string
}

export interface CompleteSprintPayload {
  next_sprint_id?: string
}

export const sprintsService = {
  list: (projectKey: string): Promise<Sprint[]> =>
    $fetch(`/projects/${projectKey}/sprints`),

  create: (projectKey: string, payload: CreateSprintPayload): Promise<Sprint> =>
    $fetch(`/projects/${projectKey}/sprints`, { method: 'POST', body: payload }),

  get: (projectKey: string, id: string): Promise<Sprint> =>
    $fetch(`/projects/${projectKey}/sprints/${id}`),

  update: (projectKey: string, id: string, payload: UpdateSprintPayload): Promise<Sprint> =>
    $fetch(`/projects/${projectKey}/sprints/${id}`, { method: 'PUT', body: payload }),

  delete: (projectKey: string, id: string): Promise<void> =>
    $fetch(`/projects/${projectKey}/sprints/${id}`, { method: 'DELETE' }),

  start: (projectKey: string, id: string): Promise<Sprint> =>
    $fetch(`/projects/${projectKey}/sprints/${id}/start`, { method: 'POST' }),

  complete: (projectKey: string, id: string, payload: CompleteSprintPayload): Promise<Sprint> =>
    $fetch(`/projects/${projectKey}/sprints/${id}/complete`, { method: 'POST', body: payload }),

  burndown: (projectKey: string, id: string): Promise<BurndownData> =>
    $fetch(`/projects/${projectKey}/sprints/${id}/burndown`),
}

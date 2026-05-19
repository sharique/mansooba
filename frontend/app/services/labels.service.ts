import type { Label } from '~/types/domain.types'

export const labelsService = {
  list(projectKey: string): Promise<Label[]> {
    const { $api } = useNuxtApp()
    return $api<Label[]>(`/projects/${projectKey}/labels`)
  },

  create(projectKey: string, name: string, color: string): Promise<Label> {
    const { $api } = useNuxtApp()
    return $api<Label>(`/projects/${projectKey}/labels`, { method: 'POST', body: { name, color } })
  },

  delete(projectKey: string, labelId: number): Promise<void> {
    const { $api } = useNuxtApp()
    return $api(`/projects/${projectKey}/labels/${labelId}`, { method: 'DELETE' })
  },

  attach(issueId: number, labelId: number): Promise<void> {
    const { $api } = useNuxtApp()
    return $api(`/issues/${issueId}/labels/${labelId}`, { method: 'POST' })
  },

  detach(issueId: number, labelId: number): Promise<void> {
    const { $api } = useNuxtApp()
    return $api(`/issues/${issueId}/labels/${labelId}`, { method: 'DELETE' })
  },
}

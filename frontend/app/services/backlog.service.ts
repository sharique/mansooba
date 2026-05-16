import type { Issue } from '~/types/domain.types'

export const backlogService = {
  getBacklog(projectKey: string): Promise<Issue[]> {
    const { $api } = useNuxtApp()
    return $api<Issue[]>(`/projects/${projectKey}/backlog`)
  },
}

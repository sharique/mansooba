import type { ActivityEvent } from '~/types/domain.types'

export const activityService = {
  listByIssue(issueId: number): Promise<ActivityEvent[]> {
    const { $api } = useNuxtApp()
    return $api<ActivityEvent[]>(`/issues/${issueId}/activity`)
  },
}

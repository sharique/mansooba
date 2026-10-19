import type { ActivityScope, OpenTask, OverviewList, RecentActivityItem, SprintSummary } from '~/types/domain.types'

export const overviewService = {
  async getSprint(projectKey: string): Promise<SprintSummary | null> {
    const { $api } = useNuxtApp()
    const res = await $api<{ sprint: SprintSummary | null }>(`/projects/${projectKey}/overview/sprint`)
    return res.sprint
  },

  getTasks(projectKey: string): Promise<OverviewList<OpenTask>> {
    const { $api } = useNuxtApp()
    return $api<OverviewList<OpenTask>>(`/projects/${projectKey}/overview/tasks`)
  },

  getActivity(projectKey: string, scope: ActivityScope): Promise<OverviewList<RecentActivityItem>> {
    const { $api } = useNuxtApp()
    return $api<OverviewList<RecentActivityItem>>(`/projects/${projectKey}/overview/activity`, { query: { scope } })
  },
}

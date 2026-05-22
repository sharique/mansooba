import type { VelocityDataPoint } from '~/types/domain.types'

export const reportsService = {
  /**
   * Fetches velocity data (committed vs. completed story points) for all
   * completed sprints in the given project, ordered oldest-first.
   */
  async getVelocity(projectKey: string): Promise<VelocityDataPoint[]> {
    const { $api } = useNuxtApp()
    return $api<VelocityDataPoint[]>(`/projects/${projectKey}/velocity`)
  },
}

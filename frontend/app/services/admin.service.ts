import type { MemberResponse, ProjectWithoutAdmin, UsernameCheckResponse } from '~/types/domain.types'

export const adminService = {
  /** Asks whether a username could be used (administrators only, read-only). */
  checkUsername(username: string): Promise<UsernameCheckResponse> {
    const { $api } = useNuxtApp()
    return $api<UsernameCheckResponse>('/admin/usernames/check', { query: { username } })
  },

  /** Projects with no active admin (system administrators only). */
  async listProjectsWithoutAdmin(): Promise<ProjectWithoutAdmin[]> {
    const { $api } = useNuxtApp()
    return (await $api<ProjectWithoutAdmin[]>('/admin/projects/without-admin')) ?? []
  },

  /** A system administrator takes the admin role in an abandoned project. */
  claimProjectAdmin(key: string): Promise<MemberResponse> {
    const { $api } = useNuxtApp()
    return $api<MemberResponse>(`/admin/projects/${key}/claim-admin`, { method: 'POST' })
  },
}

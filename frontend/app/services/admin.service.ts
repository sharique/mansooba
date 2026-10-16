import type { UsernameCheckResponse } from '~/types/domain.types'

export const adminService = {
  /** Asks whether a username could be used (administrators only, read-only). */
  checkUsername(username: string): Promise<UsernameCheckResponse> {
    const { $api } = useNuxtApp()
    return $api<UsernameCheckResponse>('/admin/usernames/check', { query: { username } })
  },
}

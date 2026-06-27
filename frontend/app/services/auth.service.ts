import { useAuthStore } from '~/stores/auth.store'
import type { AuthResponse } from '~/types/auth.types'
import type { UserProfileResponse, UpdateProfilePatch, ActivityEvent, Issue } from '~/types/domain.types'

export const authService = {
  async login(email: string, password: string): Promise<AuthResponse> {
    const { $api } = useNuxtApp()
    const data = await $api<AuthResponse>('/auth/login', {
      method: 'POST',
      body: { email, password },
    })
    useAuthStore().setAuth(data.user, data.access_token)
    return data
  },

  async register(email: string, password: string, fullName: string): Promise<AuthResponse> {
    const { $api } = useNuxtApp()
    // Do NOT call setAuth here — register is an admin action that creates another
    // user's account. Overwriting the auth store would immediately sign the admin out.
    return $api<AuthResponse>('/auth/register', {
      method: 'POST',
      body: { email, password, full_name: fullName },
    })
  },

  async logout(): Promise<void> {
    const { $api } = useNuxtApp()
    // Best-effort server-side revocation: swallow all errors so the client
    // always finishes the logout flow even if the server is unreachable.
    try {
      await $api('/auth/logout', { method: 'POST', credentials: 'include' })
    } catch {
      // intentionally ignored
    }
    useAuthStore().clearAuth()
    navigateTo('/login')
  },

  async getMe(): Promise<UserProfileResponse> {
    const { $api } = useNuxtApp()
    return $api<UserProfileResponse>('/auth/me')
  },

  async updateMe(patch: UpdateProfilePatch): Promise<UserProfileResponse> {
    const { $api } = useNuxtApp()
    return $api<UserProfileResponse>('/auth/me', {
      method: 'PUT',
      body: patch,
    })
  },

  async getMyActivity(limit = 20, offset = 0): Promise<ActivityEvent[]> {
    const { $api } = useNuxtApp()
    return $api<ActivityEvent[]>(`/auth/me/activity?limit=${limit}&offset=${offset}`)
  },

  async getMyIssues(): Promise<Issue[]> {
    const { $api } = useNuxtApp()
    return $api<Issue[]>('/auth/me/issues')
  },

  async uploadAvatar(file: File): Promise<UserProfileResponse> {
    const { $api } = useNuxtApp()
    const body = new FormData()
    body.append('avatar', file)
    return $api<UserProfileResponse>('/auth/me/avatar', { method: 'POST', body })
  },

  async deleteAvatar(): Promise<UserProfileResponse> {
    const { $api } = useNuxtApp()
    return $api<UserProfileResponse>('/auth/me/avatar', { method: 'DELETE' })
  },
}

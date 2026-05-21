import { useAuthStore } from '~/stores/auth.store'
import type { AuthResponse } from '~/types/auth.types'
import type { UserProfileResponse, UpdateProfilePatch, ActivityEvent } from '~/types/domain.types'

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
    const data = await $api<AuthResponse>('/auth/register', {
      method: 'POST',
      body: { email, password, full_name: fullName },
    })
    useAuthStore().setAuth(data.user, data.access_token)
    return data
  },

  logout(): void {
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
}

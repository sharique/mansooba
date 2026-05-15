import { useAuthStore } from '~/stores/auth.store'
import type { AuthResponse } from '~/types/auth.types'

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
}

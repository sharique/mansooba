import { defineStore } from 'pinia'
import type { User } from '~/types/domain.types'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null as User | null,
    accessToken: null as string | null,
  }),
  getters: {
    isAuthenticated: (state) => !!state.accessToken,
  },
  actions: {
    setAuth(user: User, token: string) {
      this.user = user
      this.accessToken = token
    },
    clearAuth() {
      this.user = null
      this.accessToken = null
    },
  },
  persist: true,
})

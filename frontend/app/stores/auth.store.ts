import { defineStore } from 'pinia'
import type { User, UserProfileResponse, ActivityEvent, UpdateProfilePatch } from '~/types/domain.types'
import { authService } from '~/services/auth.service'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null as User | null,
    accessToken: null as string | null,
    profile: null as UserProfileResponse | null,
    myActivity: [] as ActivityEvent[],
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
      this.profile = null
      this.myActivity = []
    },
    async fetchMe() {
      this.profile = await authService.getMe()
    },
    async updateProfile(patch: UpdateProfilePatch) {
      this.profile = await authService.updateMe(patch)
    },
    async fetchMyActivity(limit = 20, offset = 0) {
      this.myActivity = await authService.getMyActivity(limit, offset)
    },
  },
  persist: true,
})

import { useAuthStore } from '~/stores/auth.store'
import { startProfileRefresh } from '~/utils/profileRefresh'

export default defineNuxtPlugin(() => {
  const auth = useAuthStore()
  startProfileRefresh({ isAuthenticated: () => auth.isAuthenticated, fetchMe: () => auth.fetchMe() })
})

import { useAuthStore } from '~/stores/auth.store'

export default defineNuxtRouteMiddleware((to) => {
  const authStore = useAuthStore()
  const publicRoutes = ['/login', '/register']
  if (!authStore.isAuthenticated && !publicRoutes.includes(to.path)) {
    return navigateTo('/login')
  }
})

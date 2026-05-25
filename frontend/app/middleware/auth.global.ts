import { useAuthStore } from '~/stores/auth.store'

export default defineNuxtRouteMiddleware((to) => {
  const authStore = useAuthStore()
  const publicRoutes = ['/login', '/register']

  // Unauthenticated users trying to access root → send to login
  if (to.path === '/' && !authStore.isAuthenticated) {
    return navigateTo('/login', { replace: true })
  }

  // Guard protected routes
  if (!authStore.isAuthenticated && !publicRoutes.includes(to.path)) {
    return navigateTo('/login', { replace: true })
  }

  // Redirect authenticated users away from auth pages → dashboard
  if (authStore.isAuthenticated && publicRoutes.includes(to.path)) {
    return navigateTo('/', { replace: true })
  }
})

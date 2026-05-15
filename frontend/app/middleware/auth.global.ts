import { useAuthStore } from '~/stores/auth.store'

export default defineNuxtRouteMiddleware((to) => {
  const authStore = useAuthStore()
  const publicRoutes = ['/login', '/register']

  // Smart redirect for root
  if (to.path === '/') {
    return navigateTo(authStore.isAuthenticated ? '/projects' : '/login', { replace: true })
  }

  // Guard protected routes
  if (!authStore.isAuthenticated && !publicRoutes.includes(to.path)) {
    return navigateTo('/login', { replace: true })
  }

  // Redirect authenticated users away from auth pages
  if (authStore.isAuthenticated && publicRoutes.includes(to.path)) {
    return navigateTo('/projects', { replace: true })
  }
})

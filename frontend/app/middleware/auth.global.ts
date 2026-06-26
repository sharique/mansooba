import { useAuthStore } from '~/stores/auth.store'
import { useSetupStore } from '~/stores/setup.store'

export default defineNuxtRouteMiddleware(async (to) => {
  const authStore = useAuthStore()
  const setupStore = useSetupStore()

  const publicRoutes = ['/login', '/register', '/setup']

  // On every navigation, check setup status (cached after first call).
  // Must complete before any route guard logic runs per CHK028.
  let setupRequired: boolean
  try {
    setupRequired = await setupStore.checkSetupStatus()
  } catch {
    // Network error or 5xx — show full-screen error, block navigation.
    // Only allow if already on an error page to avoid redirect loops.
    if (to.path !== '/setup-error') {
      return navigateTo('/setup-error', { replace: true })
    }
    return
  }

  // Redirect to wizard when setup is required (fresh install).
  if (setupRequired && to.path !== '/setup') {
    return navigateTo('/setup', { replace: true })
  }

  // Redirect away from /setup when setup is already complete.
  if (!setupRequired && to.path === '/setup') {
    return navigateTo('/', { replace: true })
  }

  // Unauthenticated users trying to access root → send to login
  if (to.path === '/' && !authStore.isAuthenticated) {
    return navigateTo('/login', { replace: true })
  }

  // Guard protected routes
  if (!authStore.isAuthenticated && !publicRoutes.includes(to.path)) {
    return navigateTo('/login', { replace: true })
  }

  // Redirect authenticated users away from auth pages → dashboard
  if (authStore.isAuthenticated && (to.path === '/login' || to.path === '/register')) {
    return navigateTo('/', { replace: true })
  }
})

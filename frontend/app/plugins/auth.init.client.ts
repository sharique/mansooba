import { useAuthStore } from '~/stores/auth.store'

// Runs once on the client after app mounts — catches the initial page load
// that route middleware misses in SPA (ssr: false) mode.
export default defineNuxtPlugin(() => {
  const authStore = useAuthStore()
  const route = useRoute()
  const publicRoutes = ['/login', '/register']

  if (route.path === '/') {
    navigateTo(authStore.isAuthenticated ? '/projects' : '/login', { replace: true })
    return
  }

  if (!authStore.isAuthenticated && !publicRoutes.includes(route.path)) {
    navigateTo('/login', { replace: true })
  }
})

import { useAuthStore } from '~/stores/auth.store'

// Runs once on the client after app mounts — catches the initial page load
// that route middleware misses in SPA (ssr: false) mode.
export default defineNuxtPlugin(async () => {
  const authStore = useAuthStore()
  const router = useRouter()

  // Vue Router's initial navigation (parsing window.location into the real
  // route) is asynchronous. Reading useRoute() before it resolves returns
  // the router's placeholder path ('/'), not the actual URL — which meant
  // any hard-loaded deep link (e.g. clicking a /reset-password?token=...
  // link from an email client) briefly read as '/' and got redirected to
  // /login by the branch below before the real path was known.
  await router.isReady()
  const route = useRoute()
  const publicRoutes = ['/login', '/setup', '/forgot-password', '/reset-password']

  if (route.path === '/') {
    navigateTo(authStore.isAuthenticated ? '/projects' : '/login', { replace: true })
    return
  }

  if (!authStore.isAuthenticated && !publicRoutes.includes(route.path)) {
    navigateTo('/login', { replace: true })
  }
})

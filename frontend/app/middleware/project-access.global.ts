import { useAuthStore } from '~/stores/auth.store'
import { useProjectsStore } from '~/stores/projects.store'

// Someone who is not a member gets a 404 for the project (its existence is not
// revealed). Send them to a page that says so, instead of a half-empty project page.
export default defineNuxtRouteMiddleware(async (to) => {
  const match = /^\/projects\/([^/?#]+)/.exec(to.path)
  if (!match) return
  if (!useAuthStore().isAuthenticated) return

  const key = match[1]!.toLowerCase()
  const store = useProjectsStore()
  if (store.current?.key.toLowerCase() === key) return

  try {
    await store.fetchOne(key)
  }
  catch (e) {
    const status = (e as { status?: number, statusCode?: number } | null)?.status
      ?? (e as { statusCode?: number } | null)?.statusCode
    if (status === 404 || status === 403) {
      return navigateTo({ path: '/no-access', query: { project: key } }, { replace: true })
    }
    // Any other failure: let the page load and show its own error.
  }
})

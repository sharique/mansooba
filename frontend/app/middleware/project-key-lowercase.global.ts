import { lowercaseProjectKeyPath } from '~/utils/projectKeyPath'

// Keep project keys lower-case in every URL, whatever produced the link.
export default defineNuxtRouteMiddleware((to) => {
  const fixed = lowercaseProjectKeyPath(to.fullPath)
  if (fixed) return navigateTo(fixed, { replace: true })
})

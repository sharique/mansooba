import { useAuthStore } from '~/stores/auth.store'

export default defineNuxtPlugin(() => {
  const authStore = useAuthStore()
  const config = useRuntimeConfig()

  const api = $fetch.create({
    baseURL: config.public.apiBaseUrl as string,
    onRequest({ options }) {
      if (authStore.accessToken) {
        const headers = new Headers(options.headers as HeadersInit)
        headers.set('Authorization', `Bearer ${authStore.accessToken}`)
        options.headers = headers
      }
    },
    async onResponseError({ response }) {
      if (response.status === 401) {
        try {
          const { access_token } = await $fetch<{ access_token: string }>(
            '/auth/refresh',
            { baseURL: config.public.apiBaseUrl as string, method: 'POST' },
          )
          authStore.accessToken = access_token
        }
        catch {
          authStore.clearAuth()
          await navigateTo('/login')
        }
      }
    },
  })

  return { provide: { api } }
})

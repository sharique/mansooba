import { useAuthStore } from '~/stores/auth.store'

export default defineNuxtPlugin(() => {
  const authStore = useAuthStore()
  const config = useRuntimeConfig()

  const api = $fetch.create({
    baseURL: config.public.apiBaseUrl as string,
    onRequest({ options }) {
      if (authStore.accessToken) {
        options.headers = {
          ...options.headers,
          Authorization: `Bearer ${authStore.accessToken}`,
        }
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

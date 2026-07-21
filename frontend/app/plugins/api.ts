import { defineNuxtPlugin, navigateTo, useRuntimeConfig } from '#app'
import { useAuthStore } from '~/stores/auth.store'
import { useDbWaking } from '~/composables/useDbWaking'

// FR-013: the client gives up and shows a clear failure after 5 cumulative
// minutes of retrying a waking_up response.
export const WAKING_UP_MAX_RETRY_MS = 5 * 60 * 1000

interface WakingUpBody {
  status: 'waking_up'
  retry_after_ms: number
}

// Recognizes the exact wake-up-in-progress contract (contracts/wake-response.md).
// A plain 503 without this shape (e.g. the give-up response once
// RDS_START_FAILURE_BOUND is exceeded, or an unrelated infra failure) is NOT
// this contract and must not trigger a retry.
export function isWakingUpResponse(data: unknown): data is WakingUpBody {
  return (
    !!data &&
    typeof data === 'object' &&
    (data as Record<string, unknown>).status === 'waking_up' &&
    typeof (data as Record<string, unknown>).retry_after_ms === 'number'
  )
}

interface FetchWithWakeRetryOptions {
  onWaking?: (waking: boolean) => void
  sleep?: (ms: number) => Promise<void>
  now?: () => number
  maxRetryMs?: number
}

const defaultSleep = (ms: number) => new Promise<void>(resolve => setTimeout(resolve, ms))

/**
 * Wraps a fetch call with the wake-on-hit retry contract: on a waking_up
 * response, wait retry_after_ms and try again — showing a loading
 * indicator — until it succeeds or maxRetryMs (FR-013, default 5 minutes)
 * is exceeded, at which point it gives up and rethrows. Any other error
 * (including the plain 503 given once the server-side start-failure bound
 * is exceeded) propagates immediately, unretried.
 *
 * Pure and Nuxt-free so it's directly unit-testable (app/plugins/api.test.ts)
 * without a Nuxt runtime.
 */
export async function fetchWithWakeRetry<T>(
  doFetch: () => Promise<T>,
  options: FetchWithWakeRetryOptions = {},
): Promise<T> {
  const sleep = options.sleep ?? defaultSleep
  const now = options.now ?? Date.now
  const maxRetryMs = options.maxRetryMs ?? WAKING_UP_MAX_RETRY_MS
  const start = now()

  for (;;) {
    try {
      const result = await doFetch()
      options.onWaking?.(false)
      return result
    }
    catch (err) {
      const data = (err as { data?: unknown } | undefined)?.data
      if (!isWakingUpResponse(data)) {
        options.onWaking?.(false)
        throw err
      }

      options.onWaking?.(true)
      if (now() - start >= maxRetryMs) {
        options.onWaking?.(false)
        throw err
      }
      await sleep(data.retry_after_ms)
    }
  }
}

export default defineNuxtPlugin(() => {
  const authStore = useAuthStore()
  const config = useRuntimeConfig()
  const { setDbWaking } = useDbWaking()

  const rawFetch = $fetch.create({
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

  const api = ((request: Parameters<typeof rawFetch>[0], opts?: Parameters<typeof rawFetch>[1]) =>
    fetchWithWakeRetry(() => rawFetch(request, opts), { onWaking: setDbWaking })) as typeof rawFetch

  return { provide: { api } }
})

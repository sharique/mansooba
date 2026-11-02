import { describe, expect, test, vi } from 'vitest'

// api.ts's default export is a Nuxt plugin (calls defineNuxtPlugin at module
// top-level); mocking it as an identity function means the plugin body never
// actually runs on import, so this test file can exercise the pure,
// Nuxt-free fetchWithWakeRetry/isWakingUpResponse exports in isolation.
vi.mock('#app', () => ({
  defineNuxtPlugin: (fn: unknown) => fn,
  navigateTo: vi.fn(),
  useRuntimeConfig: vi.fn(),
}))
vi.mock('~/stores/auth.store', () => ({ useAuthStore: vi.fn() }))
vi.mock('~/composables/useDbWaking', () => ({ useDbWaking: vi.fn(() => ({ setDbWaking: vi.fn() })) }))

import { fetchWithWakeRetry, handleWriteDenied, isWakingUpResponse } from './api'

function wakingUpError(retryAfterMs = 5000) {
  return { data: { status: 'waking_up', retry_after_ms: retryAfterMs } }
}

function plainServiceUnavailableError() {
  return { data: { code: 'Service Unavailable', message: 'database is currently unavailable, please try again later' } }
}

describe('isWakingUpResponse', () => {
  test('recognizes the exact waking_up shape', () => {
    expect(isWakingUpResponse({ status: 'waking_up', retry_after_ms: 5000 })).toBe(true)
  })

  test('rejects a plain 503 body without the waking_up status', () => {
    expect(isWakingUpResponse({ code: 'Service Unavailable', message: 'nope' })).toBe(false)
  })

  test('rejects null/undefined/non-object values', () => {
    expect(isWakingUpResponse(null)).toBe(false)
    expect(isWakingUpResponse(undefined)).toBe(false)
    expect(isWakingUpResponse('waking_up')).toBe(false)
  })
})

describe('fetchWithWakeRetry', () => {
  test('returns the result immediately on success, with no retry and no waking indicator', async () => {
    const onWaking = vi.fn()
    const doFetch = vi.fn().mockResolvedValue({ ok: true })

    const result = await fetchWithWakeRetry(doFetch, { onWaking, sleep: vi.fn().mockResolvedValue(undefined) })

    expect(result).toEqual({ ok: true })
    expect(doFetch).toHaveBeenCalledTimes(1)
    expect(onWaking).toHaveBeenLastCalledWith(false)
  })

  test('retries after receiving a waking_up response, using its retry_after_ms, and shows the loading indicator', async () => {
    const onWaking = vi.fn()
    const sleep = vi.fn().mockResolvedValue(undefined)
    const doFetch = vi
      .fn()
      .mockRejectedValueOnce(wakingUpError(1234))
      .mockResolvedValueOnce({ ok: true })

    const result = await fetchWithWakeRetry(doFetch, { onWaking, sleep })

    expect(result).toEqual({ ok: true })
    expect(doFetch).toHaveBeenCalledTimes(2)
    expect(sleep).toHaveBeenCalledWith(1234)
    expect(onWaking).toHaveBeenCalledWith(true)
    expect(onWaking).toHaveBeenLastCalledWith(false) // cleared once it succeeds
  })

  test('gives up and surfaces a clear failure after the cumulative 5-minute bound (FR-013)', async () => {
    const onWaking = vi.fn()
    const sleep = vi.fn().mockResolvedValue(undefined)
    const doFetch = vi.fn().mockRejectedValue(wakingUpError(60_000)) // 1 minute per retry

    let elapsed = 0
    const now = vi.fn(() => elapsed)
    // Advance the fake clock by retry_after_ms each time sleep is "awaited".
    sleep.mockImplementation(async (ms: number) => {
      elapsed += ms
    })

    await expect(
      fetchWithWakeRetry(doFetch, { onWaking, sleep, now, maxRetryMs: 5 * 60 * 1000 }),
    ).rejects.toBeTruthy()

    expect(onWaking).toHaveBeenLastCalledWith(false) // indicator cleared on giving up, not left stuck on
    // 5 minutes / 1 minute per retry = 5 retries before giving up.
    expect(doFetch.mock.calls.length).toBeLessThanOrEqual(6)
    expect(doFetch.mock.calls.length).toBeGreaterThanOrEqual(5)
  })

  test('does not retry a plain 503 without the waking_up body — propagates immediately as a normal error', async () => {
    const onWaking = vi.fn()
    const sleep = vi.fn()
    const doFetch = vi.fn().mockRejectedValue(plainServiceUnavailableError())

    await expect(fetchWithWakeRetry(doFetch, { onWaking, sleep })).rejects.toBeTruthy()

    expect(doFetch).toHaveBeenCalledTimes(1)
    expect(sleep).not.toHaveBeenCalled()
    expect(onWaking).toHaveBeenLastCalledWith(false)
  })
})

describe('handleWriteDenied', () => {
  const deps = () => ({ currentKey: 'apol', refreshPermissions: vi.fn().mockResolvedValue(undefined), showError: vi.fn() })

  test('a refused change shows the server reason and refreshes the open project', async () => {
    const d = deps()
    await handleWriteDenied(403, 'PUT', "Viewers can't edit tasks", d)
    expect(d.showError).toHaveBeenCalledWith("Viewers can't edit tasks")
    expect(d.refreshPermissions).toHaveBeenCalledWith('apol')
  })

  test('a 404 on a change also refreshes (access may be gone)', async () => {
    const d = deps()
    await handleWriteDenied(404, 'DELETE', 'not found', d)
    expect(d.refreshPermissions).toHaveBeenCalledWith('apol')
  })

  test('a refused read does nothing', async () => {
    const d = deps()
    await handleWriteDenied(403, 'GET', 'nope', d)
    await handleWriteDenied(404, undefined, 'nope', d)
    expect(d.showError).not.toHaveBeenCalled()
    expect(d.refreshPermissions).not.toHaveBeenCalled()
  })

  test('other statuses do nothing', async () => {
    const d = deps()
    await handleWriteDenied(409, 'POST', 'conflict', d)
    expect(d.showError).not.toHaveBeenCalled()
  })

  test('without an open project it still explains but refreshes nothing', async () => {
    const d = { ...deps(), currentKey: undefined }
    await handleWriteDenied(403, 'POST', 'no', d)
    expect(d.showError).toHaveBeenCalledWith('no')
    expect(d.refreshPermissions).not.toHaveBeenCalled()
  })
})

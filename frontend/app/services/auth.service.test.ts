import { setActivePinia, createPinia } from 'pinia'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { useAuthStore } from '~/stores/auth.store'

const mockApi = vi.fn()

vi.stubGlobal('useNuxtApp', () => ({ $api: mockApi }))
vi.stubGlobal('navigateTo', vi.fn())

describe('authService', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockApi.mockReset()
  })

  test('login stores auth on success', async () => {
    const { authService } = await import('./auth.service')
    mockApi.mockResolvedValueOnce({
      access_token: 'tok',
      user: { id: 1, name: 'Alice', email: 'a@b.com' },
    })

    await authService.login('a@b.com', 'password1')

    const store = useAuthStore()
    expect(store.isAuthenticated).toBe(true)
    expect(store.accessToken).toBe('tok')
  })

  test('login propagates error on failure', async () => {
    const { authService } = await import('./auth.service')
    mockApi.mockRejectedValueOnce(new Error('Unauthorized'))

    await expect(authService.login('a@b.com', 'wrong')).rejects.toThrow('Unauthorized')
    expect(useAuthStore().isAuthenticated).toBe(false)
  })
})

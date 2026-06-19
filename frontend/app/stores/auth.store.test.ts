import { setActivePinia, createPinia } from 'pinia'
import { beforeEach, describe, expect, test, it, vi } from 'vitest'
import { useAuthStore } from './auth.store'
import { authService } from '~/services/auth.service'
import type { UserProfileResponse } from '~/types/domain.types'

vi.mock('~/services/auth.service', () => ({
  authService: {
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
    getMe: vi.fn(),
    updateMe: vi.fn(),
    getMyActivity: vi.fn(),
    getMyIssues: vi.fn(),
  },
}))

describe('auth store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  test('initial state is unauthenticated', () => {
    const store = useAuthStore()
    expect(store.isAuthenticated).toBe(false)
    expect(store.user).toBeNull()
    expect(store.accessToken).toBeNull()
  })

  test('setAuth stores user and token', () => {
    const store = useAuthStore()
    store.setAuth({ id: 1, name: 'Alice', email: 'a@b.com' }, 'tok123')
    expect(store.isAuthenticated).toBe(true)
    expect(store.accessToken).toBe('tok123')
    expect(store.user?.name).toBe('Alice')
  })

  test('clearAuth resets state', () => {
    const store = useAuthStore()
    store.setAuth({ id: 1, name: 'Alice', email: 'a@b.com' }, 'tok123')
    store.clearAuth()
    expect(store.isAuthenticated).toBe(false)
    expect(store.user).toBeNull()
    expect(store.accessToken).toBeNull()
    expect(store.profile).toBeNull()
    expect(store.myActivity).toHaveLength(0)
  })

  it('fetchMe sets profile state', async () => {
    const mockProfile: UserProfileResponse = {
      id: 1, name: 'Alice', email: 'alice@example.com',
      avatar_url: '', timezone: 'UTC', is_admin: false, created_at: '',
    }
    vi.mocked(authService.getMe).mockResolvedValue(mockProfile)

    const store = useAuthStore()
    await store.fetchMe()
    expect(store.profile?.name).toBe('Alice')
  })

  it('updateProfile updates profile state', async () => {
    const updated: UserProfileResponse = {
      id: 1, name: 'Alice B', email: 'alice@example.com',
      avatar_url: '', timezone: 'America/New_York', is_admin: false, created_at: '',
    }
    vi.mocked(authService.updateMe).mockResolvedValue(updated)

    const store = useAuthStore()
    await store.updateProfile({ full_name: 'Alice B', timezone: 'America/New_York' })
    expect(store.profile?.name).toBe('Alice B')
    expect(store.profile?.timezone).toBe('America/New_York')
  })
})

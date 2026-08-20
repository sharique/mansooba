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
    changePassword: vi.fn(),
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

  // T013 (012-change-password): will not compile/run until
  // useAuthStore().changePassword and authService.changePassword exist —
  // the expected TDD red state. See
  // specs/012-change-password/IMPLEMENTATION_GUIDE.md.
  it('changePassword delegates to authService.changePassword with both passwords', async () => {
    vi.mocked(authService.changePassword).mockResolvedValue(undefined)

    const store = useAuthStore()
    await store.changePassword('OldPassword1', 'NewPassword2')

    expect(authService.changePassword).toHaveBeenCalledWith('OldPassword1', 'NewPassword2')
  })
})

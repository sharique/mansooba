import { setActivePinia, createPinia } from 'pinia'
import { beforeEach, describe, expect, test } from 'vitest'
import { useAuthStore } from './auth.store'

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
  })
})

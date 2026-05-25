import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useThemeStore } from '~/stores/theme.store'

describe('useThemeStore', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('defaults to null (follow system)', () => {
    expect(useThemeStore().selected).toBeNull()
  })

  it('setTheme stores the chosen theme', () => {
    const store = useThemeStore()
    store.setTheme('mansooba-dark')
    expect(store.selected).toBe('mansooba-dark')
  })

  it('toggle flips between light and dark, seeding from a default', () => {
    const store = useThemeStore()
    store.toggle('mansooba')            // current effective theme passed in
    expect(store.selected).toBe('mansooba-dark')
    store.toggle('mansooba-dark')
    expect(store.selected).toBe('mansooba')
  })

  it('toggle uses selected over the passed-in current when selected is already set', () => {
    const store = useThemeStore()
    store.setTheme('mansooba-dark')
    // even though we pass 'mansooba' as current, selected ('mansooba-dark') takes precedence
    store.toggle('mansooba')
    expect(store.selected).toBe('mansooba')
  })
})

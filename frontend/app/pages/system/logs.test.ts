// @vitest-environment happy-dom
import { flushPromises, shallowMount } from '@vue/test-utils'
import { ref } from 'vue'
import { beforeEach, describe, it, expect, vi } from 'vitest'
import LogsPage from './logs.vue'

const mockNavigateTo = vi.fn()
vi.stubGlobal('navigateTo', mockNavigateTo)

const mockAuthStore = { isAdmin: true }
vi.mock('~/stores/auth.store', () => ({
  useAuthStore: () => mockAuthStore,
}))

const mockFetchLogs = vi.fn().mockResolvedValue(undefined)
// Real Vue refs (not plain objects) — a plain `{ value: null }` is always
// truthy in a v-if, which would break the error/empty-state assertions below.
const entriesRef = ref<unknown[]>([])
const totalRef = ref(0)
const pageRef = ref(1)
const loadingRef = ref(false)
const errorRef = ref<string | null>(null)

vi.mock('~/composables/useSystemLogs', () => ({
  useSystemLogs: () => ({
    entries: entriesRef,
    total: totalRef,
    page: pageRef,
    loading: loadingRef,
    error: errorRef,
    fetchLogs: mockFetchLogs,
  }),
}))

const globalStubs = { Icon: true }

describe('system/logs page', () => {
  beforeEach(() => {
    mockAuthStore.isAdmin = true
    mockNavigateTo.mockReset()
    mockFetchLogs.mockReset().mockResolvedValue(undefined)
    entriesRef.value = []
    totalRef.value = 0
    pageRef.value = 1
    loadingRef.value = false
    errorRef.value = null
  })

  it('redirects non-admins away and never fetches logs', async () => {
    mockAuthStore.isAdmin = false
    shallowMount(LogsPage, { global: { stubs: globalStubs } })
    await flushPromises()
    expect(mockNavigateTo).toHaveBeenCalledWith('/')
    expect(mockFetchLogs).not.toHaveBeenCalled()
  })

  it('shows a loading skeleton while fetching, not the empty state', async () => {
    loadingRef.value = true
    const w = shallowMount(LogsPage, { global: { stubs: globalStubs } })
    await flushPromises()
    expect(w.find('.skeleton').exists()).toBe(true)
    expect(w.text()).not.toContain('No log entries match')
  })

  it('shows the explicit empty state when not loading and there are zero entries', async () => {
    loadingRef.value = false
    entriesRef.value = []
    const w = shallowMount(LogsPage, { global: { stubs: globalStubs } })
    await flushPromises()
    expect(w.text()).toContain('No log entries match the current filters.')
    expect(w.find('table').exists()).toBe(false)
  })

  it('renders a table row per entry when entries are present', async () => {
    loadingRef.value = false
    entriesRef.value = [
      {
        id: '1-abc',
        event_category: 'authentication',
        action: 'login_failed',
        outcome: 'failure',
        actor_label: 'jane@example.com',
        target_label: null,
        detail: 'invalid credentials',
        created_at: '2026-07-26T14:00:00Z',
      },
    ]
    totalRef.value = 1
    const w = shallowMount(LogsPage, { global: { stubs: globalStubs } })
    await flushPromises()
    expect(w.find('table').exists()).toBe(true)
    expect(w.findAll('tbody tr')).toHaveLength(1)
    expect(w.text()).toContain('jane@example.com')
  })

  it('fetches logs on mount for an admin', async () => {
    shallowMount(LogsPage, { global: { stubs: globalStubs } })
    await flushPromises()
    expect(mockFetchLogs).toHaveBeenCalledWith({}, 1, 20)
  })
})

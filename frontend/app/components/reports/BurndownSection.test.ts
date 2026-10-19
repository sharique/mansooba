// @vitest-environment happy-dom
import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, it, expect, vi } from 'vitest'
import BurndownSection from './BurndownSection.vue'
import type { Sprint } from '~/types/domain.types'

const mockFetchSprints = vi.fn()
const mockFetchBurndown = vi.fn()
const store = {
  sprints: [] as Sprint[],
  activeSprint: null as Sprint | null,
  burndownData: null as unknown,
  fetchSprints: mockFetchSprints,
  fetchBurndown: mockFetchBurndown,
}

vi.stubGlobal('useSprintsStore', () => store)
vi.stubGlobal('useTimeFormatter', () => ({ formatDate: (d: string | null) => (d ? `fmt(${d})` : '—') }))

// The API sends numeric ids even though the type says string.
const mk = (id: number, name: string, status: string, start: string): Sprint =>
  ({ id: id as unknown as string, name, status, start_date: `${start}T00:00:00Z`, end_date: '2026-05-30T00:00:00Z' }) as Sprint
const done = mk(1, 'Sprint 1', 'completed', '2026-04-01')
const current = mk(2, 'Sprint 2', 'active', '2026-05-01')
const planned = mk(3, 'Sprint 3', 'planning', '2026-06-01')

const mount = () => shallowMount(BurndownSection, { props: { projectKey: 'TEST' } })
const options = (w: ReturnType<typeof mount>) => w.findAll('option').map(o => o.text())

describe('BurndownSection', () => {
  beforeEach(() => {
    store.sprints = []
    store.activeSprint = null
    store.burndownData = null
    mockFetchSprints.mockReset().mockResolvedValue([])
    mockFetchBurndown.mockReset().mockResolvedValue({})
  })

  it('defaults to the sprint in progress and loads its burndown', async () => {
    store.sprints = [done, current, planned]
    store.activeSprint = current
    store.burndownData = { sprint_name: 'Sprint 2' }
    const w = mount()
    await flushPromises()
    expect(mockFetchBurndown).toHaveBeenCalledWith('TEST', '2')
    expect((w.find('select').element as HTMLSelectElement).value).toBe('2')
    expect(w.text()).toContain('fmt(2026-05-01T00:00:00Z) → fmt(2026-05-30T00:00:00Z)')
    expect(w.find('chartsburndownchart').exists()).toBe(true)
  })

  it('offers None plus started sprints, newest first, marking the current one', async () => {
    store.sprints = [done, current, planned]
    store.activeSprint = current
    const w = mount()
    await flushPromises()
    expect(options(w)).toEqual(['None', 'Sprint 2 (current)', 'Sprint 1'])
  })

  it('selects None and loads nothing when no sprint is in progress', async () => {
    store.sprints = [done, planned]
    const w = mount()
    await flushPromises()
    expect(mockFetchBurndown).not.toHaveBeenCalled()
    expect((w.find('select').element as HTMLSelectElement).value).toBe('')
    expect(w.find('chartsburndownchart').exists()).toBe(false)
    expect(w.text()).toContain('No sprint is in progress')
  })

  it('loads another sprint when it is picked, and clears the chart for None', async () => {
    store.sprints = [done, current]
    store.activeSprint = current
    store.burndownData = { sprint_name: 'x' }
    const w = mount()
    await flushPromises()

    await w.find('select').setValue('1')
    await flushPromises()
    expect(mockFetchBurndown).toHaveBeenLastCalledWith('TEST', '1')
    expect(w.text()).toContain('fmt(2026-04-01T00:00:00Z)')

    mockFetchBurndown.mockClear()
    await w.find('select').setValue('')
    await flushPromises()
    expect(mockFetchBurndown).not.toHaveBeenCalled()
    expect(w.find('chartsburndownchart').exists()).toBe(false)
  })

  it('shows an error inside the section when the burndown fails', async () => {
    store.sprints = [current]
    store.activeSprint = current
    mockFetchBurndown.mockRejectedValue(new Error('boom'))
    const w = mount()
    await flushPromises()
    expect(w.text()).toContain('Failed to load burndown')
    expect(w.find('select').exists()).toBe(true)
  })

  it('shows an error when sprints cannot be loaded', async () => {
    mockFetchSprints.mockRejectedValue(new Error('down'))
    const w = mount()
    await flushPromises()
    expect(w.text()).toContain('Failed to load burndown')
  })
})

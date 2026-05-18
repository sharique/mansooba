// @vitest-environment happy-dom
import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, it, expect, vi } from 'vitest'
import BoardPage from './board.vue'
import type { Sprint } from '~/types/domain.types'

vi.mock('~/services/board.service', () => ({
  boardService: { getBoard: vi.fn().mockResolvedValue({ columns: [] }) },
}))

const mockFetchSprints = vi.fn().mockResolvedValue([])
const mockFetchBurndown = vi.fn().mockResolvedValue({})
const mockSprintsStore = {
  activeSprint: null as Sprint | null,
  sprints: [] as Sprint[],
  burndownData: null,
  fetchSprints: mockFetchSprints,
  fetchBurndown: mockFetchBurndown,
}

vi.stubGlobal('useRoute', () => ({ params: { key: 'TEST' } }))
vi.stubGlobal('useIssuesStore', () => ({ update: vi.fn() }))
vi.stubGlobal('useSprintsStore', () => mockSprintsStore)
vi.stubGlobal('useToast', () => ({ showSuccess: vi.fn(), showError: vi.fn() }))

const activeSprint: Sprint = {
  id: 'sprint-active',
  project_id: 'p-1',
  name: 'Sprint 1',
  goal: '',
  status: 'active',
  start_date: '2026-05-01',
  end_date: '2026-05-14',
  created_at: '2026-05-01T00:00:00Z',
  updated_at: '2026-05-01T00:00:00Z',
}

describe('board page', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockSprintsStore.activeSprint = null
    mockSprintsStore.burndownData = null
    mockFetchSprints.mockReset().mockResolvedValue([])
    mockFetchBurndown.mockReset().mockResolvedValue({})
  })

  it('does not show burndown section when no active sprint', async () => {
    const w = shallowMount(BoardPage)
    await flushPromises()
    expect(w.find('section.mb-6').exists()).toBe(false)
  })

  it('shows burndown section when active sprint is set', async () => {
    mockSprintsStore.activeSprint = activeSprint
    const w = shallowMount(BoardPage)
    await flushPromises()
    expect(w.find('section.mb-6').exists()).toBe(true)
    expect(w.text()).toContain('Sprint 1')
  })

  it('shows "Backlog →" navigation link', async () => {
    const w = shallowMount(BoardPage)
    await flushPromises()
    expect(w.text()).toContain('Backlog →')
  })

  it('calls fetchSprints on mount', async () => {
    shallowMount(BoardPage)
    await flushPromises()
    expect(mockFetchSprints).toHaveBeenCalledWith('TEST')
  })
})

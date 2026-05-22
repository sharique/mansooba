// @vitest-environment happy-dom
import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, it, expect, vi } from 'vitest'
import BacklogPage from './backlog.vue'
import { backlogService } from '~/services/backlog.service'

vi.mock('~/services/backlog.service', () => ({
  backlogService: { getBacklog: vi.fn().mockResolvedValue([]) },
}))
vi.mock('~/services/projects.service', () => ({
  projectsService: { listMembers: vi.fn().mockResolvedValue([]) },
}))
vi.mock('~/services/issues.service', () => ({
  issuesService: { update: vi.fn() },
}))

const mockFetchSprints = vi.fn().mockResolvedValue([])
const mockFetchOne = vi.fn().mockResolvedValue({})

vi.stubGlobal('useRoute', () => ({ params: { key: 'TEST' } }))
vi.stubGlobal('useAuthStore', () => ({ user: { id: 1 } }))
vi.stubGlobal('useProjectsStore', () => ({ current: null, fetchOne: mockFetchOne }))
vi.stubGlobal('useSprintsStore', () => ({ sprints: [], fetchSprints: mockFetchSprints }))
vi.stubGlobal('useIssuesStore', () => ({ searchResults: [], searchIssues: vi.fn().mockResolvedValue(undefined) }))
vi.stubGlobal('useToast', () => ({ showSuccess: vi.fn(), showError: vi.fn() }))

describe('backlog page', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockFetchSprints.mockReset().mockResolvedValue([])
    mockFetchOne.mockReset().mockResolvedValue({})
    vi.mocked(backlogService.getBacklog).mockResolvedValue([])
  })

  it('renders without error on successful load', async () => {
    const w = shallowMount(BacklogPage)
    await flushPromises()
    expect(w.find('.alert.alert-error').exists()).toBe(false)
  })

  it('calls fetchSprints on mount', async () => {
    shallowMount(BacklogPage)
    await flushPromises()
    expect(mockFetchSprints).toHaveBeenCalledWith('TEST')
  })

  it('shows sprint management and backlog sections', async () => {
    const w = shallowMount(BacklogPage)
    await flushPromises()
    const sections = w.findAll('section')
    expect(sections.length).toBeGreaterThanOrEqual(2)
  })

  it('shows error alert when backlog fails to load', async () => {
    vi.mocked(backlogService.getBacklog).mockRejectedValueOnce(new Error('Network error'))
    const w = shallowMount(BacklogPage)
    await flushPromises()
    expect(w.find('.alert.alert-error').exists()).toBe(true)
    expect(w.text()).toContain('Failed to load backlog')
  })
})

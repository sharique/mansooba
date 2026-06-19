// @vitest-environment happy-dom
import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, it, expect, vi } from 'vitest'
import ProjectsPage from './index.vue'

// Mock useToast globally (Nuxt auto-import)
vi.stubGlobal('useToast', () => ({ showSuccess: vi.fn(), showError: vi.fn() }))

const mockFetchAll = vi.fn().mockResolvedValue(undefined)
const mockProjectsStore = {
  projects: [] as { id: number; key: string; name: string }[],
  fetchAll: mockFetchAll,
}

// Mock the module import — index.vue imports useProjectsStore explicitly
vi.mock('~/stores/projects.store', () => ({
  useProjectsStore: () => mockProjectsStore,
}))

const globalStubs = {
  UiEmptyState: { name: 'UiEmptyState', template: '<div data-testid="empty-state"><slot name="action" /></div>' },
  ProjectsProjectCard: true,
  ProjectsProjectForm: true,
  Icon: true,
}

describe('projects index page', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockProjectsStore.projects = []
    mockFetchAll.mockReset().mockResolvedValue(undefined)
  })

  it('shows empty state when there are zero projects', async () => {
    mockProjectsStore.projects = []
    const w = shallowMount(ProjectsPage, { global: { stubs: globalStubs } })
    await flushPromises()
    expect(w.find('[data-testid="empty-state"]').exists()).toBe(true)
  })

  it('hides empty state when projects are present', async () => {
    mockProjectsStore.projects = [{ id: 1, key: 'PROJ', name: 'My Project' }]
    const w = shallowMount(ProjectsPage, { global: { stubs: globalStubs } })
    await flushPromises()
    expect(w.find('[data-testid="empty-state"]').exists()).toBe(false)
  })

  it('does not show empty state while loading', async () => {
    mockFetchAll.mockReturnValue(new Promise(() => {})) // never resolves
    const w = shallowMount(ProjectsPage, { global: { stubs: globalStubs } })
    // Before promises flush (still loading) — skeleton shows, not empty state
    expect(w.find('[data-testid="empty-state"]').exists()).toBe(false)
  })
})

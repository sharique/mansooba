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

const permissions = { can: vi.fn().mockReturnValue(true), reason: vi.fn().mockReturnValue("Viewers can't edit tasks") }
vi.stubGlobal('useProjectPermissions', () => permissions)

describe('backlog page', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockFetchSprints.mockReset().mockResolvedValue([])
    mockFetchOne.mockReset().mockResolvedValue({})
    vi.mocked(backlogService.getBacklog).mockResolvedValue([])
    permissions.can.mockReturnValue(true)
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

  it('does not render the sprint list', async () => {
    const w = shallowMount(BacklogPage)
    await flushPromises()
    expect(w.find('sprintssprintlist').exists()).toBe(false)
    expect(w.findAll('section')).toHaveLength(1)
  })

  it('still loads the project (access check) and shows an alert when that fails', async () => {
    mockFetchOne.mockRejectedValueOnce(new Error('403'))
    const w = shallowMount(BacklogPage)
    await flushPromises()
    expect(mockFetchOne).toHaveBeenCalledWith('TEST')
    expect(w.find('.alert.alert-error').exists()).toBe(true)
  })

  it('shows error alert when backlog fails to load', async () => {
    vi.mocked(backlogService.getBacklog).mockRejectedValueOnce(new Error('Network error'))
    const w = shallowMount(BacklogPage)
    await flushPromises()
    expect(w.find('.alert.alert-error').exists()).toBe(true)
    expect(w.text()).toContain('Failed to load backlog')
  })

  describe('Create Issue', () => {
    // Nuxt auto-registers components; here the dialog is a stub that just records its props.
    const mountPage = () => shallowMount(BacklogPage, {
      global: {
        stubs: {
          IssuesCreateIssueModal: {
            name: 'IssuesCreateIssueModal',
            props: ['open', 'projectKey', 'defaultStatus'],
            emits: ['created', 'close'],
            template: '<div />',
          },
        },
      },
    })
    const createButton = (w: ReturnType<typeof shallowMount>) =>
      w.findAll('button').find(b => b.text() === 'Create Issue')

    it('shows a Create Issue button next to the page title', async () => {
      const w = mountPage()
      await flushPromises()
      const btn = createButton(w)
      expect(btn?.exists()).toBe(true)
      expect(btn?.attributes('disabled')).toBeUndefined()
    })

    it('opens the new-issue dialog for this project when clicked', async () => {
      const w = mountPage()
      await flushPromises()
      const modal = () => w.findComponent({ name: 'IssuesCreateIssueModal' })
      expect(modal().props('open')).toBe(false)
      await createButton(w)!.trigger('click')
      expect(modal().props('open')).toBe(true)
      expect(modal().props('projectKey')).toBe('TEST')
    })

    it('is disabled with the reason for someone who cannot edit tasks', async () => {
      permissions.can.mockReturnValue(false)
      const w = mountPage()
      await flushPromises()
      const btn = createButton(w)
      expect(btn?.attributes('disabled')).toBeDefined()
      expect(btn?.attributes('title')).toBe("Viewers can't edit tasks")
    })

    it('reloads the backlog and closes the dialog once an issue is created', async () => {
      const w = mountPage()
      await flushPromises()
      vi.mocked(backlogService.getBacklog).mockClear()
      vi.mocked(backlogService.getBacklog).mockResolvedValue([{ id: 9, key: 'TEST-9', title: 'New one' } as never])
      await createButton(w)!.trigger('click')
      const modal = w.findComponent({ name: 'IssuesCreateIssueModal' })
      await modal.vm.$emit('created', { id: 9 })
      await flushPromises()
      expect(backlogService.getBacklog).toHaveBeenCalledWith('TEST')
      expect(modal.props('open')).toBe(false)
    })
  })
})

// @vitest-environment happy-dom
import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import SprintsPage from './sprints.vue'
import { useProjectPermissions } from '~/composables/useProjectPermissions'

const mockFetchOne = vi.fn()
const mockFetchSprints = vi.fn()
const projects = reactive({
  current: null as { myRole?: string; permissions?: string[] } | null,
  fetchOne: mockFetchOne,
})
const auth = reactive({ user: { id: 2 } })

vi.stubGlobal('useRoute', () => ({ params: { key: 'TEST' } }))
vi.stubGlobal('useAuthStore', () => auth)
vi.stubGlobal('useProjectsStore', () => projects)
vi.stubGlobal('useSprintsStore', () => ({ sprints: [], fetchSprints: mockFetchSprints }))
vi.stubGlobal('useProjectPermissions', useProjectPermissions)
vi.stubGlobal('useToast', () => ({ showSuccess: vi.fn(), showError: vi.fn() }))

describe('sprints page', () => {
  beforeEach(() => {
    mockFetchOne.mockReset().mockResolvedValue(undefined)
    mockFetchSprints.mockReset().mockResolvedValue(undefined)
    projects.current = { myRole: 'viewer', permissions: ['project.view'] }
  })

  it('fetches the project and the sprints on mount', async () => {
    shallowMount(SprintsPage)
    await flushPromises()
    expect(mockFetchOne).toHaveBeenCalledWith('TEST')
    expect(mockFetchSprints).toHaveBeenCalledWith('TEST')
  })

  it('passes can-manage true when the server grants sprint.manage', async () => {
    projects.current = { myRole: 'admin', permissions: ['project.view', 'sprint.manage'] }
    const w = shallowMount(SprintsPage)
    await flushPromises()
    expect(w.find('sprintssprintlist').attributes('can-manage')).toBe('true')
  })

  it('passes can-manage false for a member without sprint.manage', async () => {
    projects.current = { myRole: 'member', permissions: ['project.view', 'task.write'] }
    const w = shallowMount(SprintsPage)
    await flushPromises()
    expect(w.find('sprintssprintlist').attributes('can-manage')).toBe('false')
  })

  it('passes can-manage false for a viewer', async () => {
    const w = shallowMount(SprintsPage)
    await flushPromises()
    expect(w.find('sprintssprintlist').attributes('can-manage')).toBe('false')
  })

  it('shows no backlog list and no search box', async () => {
    const w = shallowMount(SprintsPage)
    await flushPromises()
    expect(w.find('backlogList').exists()).toBe(false)
    expect(w.find('issuesissuesearchbar').exists()).toBe(false)
  })

  it('shows the load-error alert when the project fetch fails', async () => {
    mockFetchOne.mockRejectedValueOnce(new Error('403'))
    const w = shallowMount(SprintsPage)
    await flushPromises()
    expect(w.find('.alert.alert-error').exists()).toBe(true)
  })
})

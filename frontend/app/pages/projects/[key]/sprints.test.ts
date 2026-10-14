// @vitest-environment happy-dom
import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import SprintsPage from './sprints.vue'
import { useCanManageProject } from '~/composables/useCanManageProject'

const mockFetchOne = vi.fn()
const mockFetchMembers = vi.fn()
const mockFetchSprints = vi.fn()
const projects = reactive({
  current: { ownerId: 1 } as { ownerId: number } | null,
  members: [] as { user_id: number; role: string }[],
  fetchOne: mockFetchOne,
  fetchMembers: mockFetchMembers,
})
const auth = reactive({ user: { id: 2 } })

vi.stubGlobal('useRoute', () => ({ params: { key: 'TEST' } }))
vi.stubGlobal('useAuthStore', () => auth)
vi.stubGlobal('useProjectsStore', () => projects)
vi.stubGlobal('useSprintsStore', () => ({ sprints: [], fetchSprints: mockFetchSprints }))
vi.stubGlobal('useCanManageProject', useCanManageProject)
vi.stubGlobal('useToast', () => ({ showSuccess: vi.fn(), showError: vi.fn() }))

describe('sprints page', () => {
  beforeEach(() => {
    mockFetchOne.mockReset().mockResolvedValue(undefined)
    mockFetchMembers.mockReset().mockResolvedValue(undefined)
    mockFetchSprints.mockReset().mockResolvedValue(undefined)
    projects.current = { ownerId: 1 }
    projects.members = []
  })

  it('fetches the project, its members and the sprints on mount', async () => {
    shallowMount(SprintsPage)
    await flushPromises()
    expect(mockFetchOne).toHaveBeenCalledWith('TEST')
    expect(mockFetchMembers).toHaveBeenCalledWith('TEST')
    expect(mockFetchSprints).toHaveBeenCalledWith('TEST')
  })

  it('passes can-manage true for an admin member', async () => {
    projects.members = [{ user_id: 2, role: 'admin' }]
    const w = shallowMount(SprintsPage)
    await flushPromises()
    expect(w.find('sprintssprintlist').attributes('can-manage')).toBe('true')
  })

  it('passes can-manage false for a regular member', async () => {
    projects.members = [{ user_id: 2, role: 'member' }]
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

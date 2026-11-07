import { setActivePinia, createPinia } from 'pinia'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { useProjectsStore } from './projects.store'

const mockList = vi.fn()
const mockGet = vi.fn()
const mockCreate = vi.fn()
const mockUpdate = vi.fn()
const mockListMembers = vi.fn()
const mockChangeRole = vi.fn()
const mockDelete = vi.fn()

vi.mock('~/services/projects.service', () => ({
  projectsService: {
    list: () => mockList(),
    get: (key: string) => mockGet(key),
    create: (data: unknown) => mockCreate(data),
    update: (key: string, data: unknown) => mockUpdate(key, data),
    listMembers: (key: string) => mockListMembers(key),
    addMember: vi.fn(),
    removeMember: vi.fn(),
    changeMemberRole: (...a: unknown[]) => mockChangeRole(...a),
    deleteProject: (...a: unknown[]) => mockDelete(...a),
  },
}))

const project = { id: 1, key: 'PROJ', name: 'My Project', description: '', ownerId: 1 }

describe('projects store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockList.mockReset()
    mockGet.mockReset()
    mockCreate.mockReset()
    mockUpdate.mockReset()
    mockListMembers.mockReset()
    mockChangeRole.mockReset()
    mockDelete.mockReset()
  })

  test('fetchAll populates projects', async () => {
    mockList.mockResolvedValueOnce([project])
    const store = useProjectsStore()
    await store.fetchAll()
    expect(store.projects).toHaveLength(1)
    const project0 = store.projects[0]
    expect(project0).toBeDefined()
    expect(project0!.key).toBe('PROJ')
  })

  test('fetchOne sets current', async () => {
    mockGet.mockResolvedValueOnce(project)
    const store = useProjectsStore()
    await store.fetchOne('PROJ')
    expect(store.current?.key).toBe('PROJ')
  })

  test('create appends project to list', async () => {
    mockCreate.mockResolvedValueOnce(project)
    const store = useProjectsStore()
    const result = await store.create({ name: 'My Project', key: 'PROJ' })
    expect(store.projects).toHaveLength(1)
    expect(result.key).toBe('PROJ')
  })

  describe('members', () => {
    const member = { user_id: 2, name: 'Priya', email: 'priya@example.com', role: 'member' }

    test('fetchMembers fills members from the service', async () => {
      mockListMembers.mockResolvedValueOnce([member])
      const store = useProjectsStore()
      await store.fetchMembers('PROJ')
      expect(mockListMembers).toHaveBeenCalledWith('PROJ')
      expect(store.members).toEqual([member])
    })

    test('a failed fetch leaves members empty and does not throw', async () => {
      mockListMembers.mockRejectedValueOnce(new Error('403'))
      const store = useProjectsStore()
      await expect(store.fetchMembers('PROJ')).resolves.toBeUndefined()
      expect(store.members).toEqual([])
    })

    test('members from another project are cleared when fetching a different key', async () => {
      mockListMembers.mockResolvedValueOnce([member])
      const store = useProjectsStore()
      await store.fetchMembers('PROJ')
      mockListMembers.mockRejectedValueOnce(new Error('403'))
      await store.fetchMembers('OTHER')
      expect(store.members).toEqual([])
    })
  })

  describe('changeMemberRole', () => {
    test('updates the member row with the server response', async () => {
      const store = useProjectsStore()
      store.members = [{ user_id: 2, name: 'P', email: 'p@x', username: 'p', role: 'member', is_active: true }]
      mockChangeRole.mockResolvedValueOnce({ user_id: 2, name: 'P', email: 'p@x', username: 'p', role: 'viewer', is_active: true })
      await store.changeMemberRole('PROJ', 2, 'viewer')
      expect(mockChangeRole).toHaveBeenCalledWith('PROJ', 2, 'viewer')
      expect(store.members[0]!.role).toBe('viewer')
    })

    test('surfaces the server message when the last admin would be demoted', async () => {
      mockChangeRole.mockRejectedValueOnce({ status: 409, data: { code: 'Conflict', message: 'A project needs at least one active admin' } })
      const store = useProjectsStore()
      await expect(store.changeMemberRole('PROJ', 2, 'viewer')).rejects.toThrow('A project needs at least one active admin')
    })
  })

  describe('deleteProject', () => {
    test('calls the service and drops the project from the list', async () => {
      mockDelete.mockResolvedValueOnce(undefined)
      const store = useProjectsStore()
      store.projects = [project]
      store.current = project
      await store.deleteProject('PROJ', 'My Project')
      expect(mockDelete).toHaveBeenCalledWith('PROJ', 'My Project')
      expect(store.projects).toEqual([])
      expect(store.current).toBeNull()
    })

    test('carries the task count on a 409', async () => {
      mockDelete.mockRejectedValueOnce({ status: 409, data: { message: 'Project still has 3 tasks. Remove them first.', task_count: 3 } })
      const store = useProjectsStore()
      store.projects = [project]
      await expect(store.deleteProject('PROJ', 'My Project')).rejects.toMatchObject({
        message: 'Project still has 3 tasks. Remove them first.', status: 409, taskCount: 3,
      })
      expect(store.projects).toHaveLength(1)
    })

    test('surfaces a wrong-name 400', async () => {
      mockDelete.mockRejectedValueOnce({ status: 400, data: { message: 'Type the project name to confirm' } })
      const store = useProjectsStore()
      await expect(store.deleteProject('PROJ', 'x')).rejects.toMatchObject({ status: 400, message: 'Type the project name to confirm' })
    })
  })

  describe('refreshPermissions', () => {
    test('refetches the project so role and permissions update', async () => {
      const store = useProjectsStore()
      store.current = { ...project, myRole: 'member', permissions: ['task.write'] }
      mockGet.mockResolvedValueOnce({ ...project, myRole: 'viewer', permissions: ['project.view'] })
      await store.refreshPermissions('PROJ')
      expect(mockGet).toHaveBeenCalledWith('PROJ')
      expect(store.current?.myRole).toBe('viewer')
      expect(store.current?.permissions).toEqual(['project.view'])
    })

    test('removes every permission when the project is no longer visible (404)', async () => {
      const store = useProjectsStore()
      store.current = { ...project, myRole: 'member', permissions: ['task.write'] }
      mockGet.mockRejectedValueOnce({ status: 404 })
      await store.refreshPermissions('PROJ')
      expect(store.current?.permissions).toEqual([])
    })

    test('keeps the current permissions on a network failure', async () => {
      const store = useProjectsStore()
      store.current = { ...project, myRole: 'member', permissions: ['task.write'] }
      mockGet.mockRejectedValueOnce(new Error('network'))
      await store.refreshPermissions('PROJ')
      expect(store.current?.permissions).toEqual(['task.write'])
    })
  })
})

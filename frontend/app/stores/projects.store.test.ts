import { setActivePinia, createPinia } from 'pinia'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { useProjectsStore } from './projects.store'

const mockList = vi.fn()
const mockGet = vi.fn()
const mockCreate = vi.fn()
const mockUpdate = vi.fn()
const mockListMembers = vi.fn()

vi.mock('~/services/projects.service', () => ({
  projectsService: {
    list: () => mockList(),
    get: (key: string) => mockGet(key),
    create: (data: unknown) => mockCreate(data),
    update: (key: string, data: unknown) => mockUpdate(key, data),
    listMembers: (key: string) => mockListMembers(key),
    addMember: vi.fn(),
    removeMember: vi.fn(),
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
})

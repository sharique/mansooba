import { setActivePinia, createPinia } from 'pinia'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { useProjectsStore } from './projects.store'

const mockList = vi.fn()
const mockGet = vi.fn()
const mockCreate = vi.fn()
const mockUpdate = vi.fn()

vi.mock('~/services/projects.service', () => ({
  projectsService: {
    list: () => mockList(),
    get: (key: string) => mockGet(key),
    create: (data: unknown) => mockCreate(data),
    update: (key: string, data: unknown) => mockUpdate(key, data),
    listMembers: vi.fn(),
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
})

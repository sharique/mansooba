import { setActivePinia, createPinia } from 'pinia'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { useIssuesStore } from './issues.store'

const mockList = vi.fn()
const mockGet = vi.fn()
const mockCreate = vi.fn()
const mockUpdate = vi.fn()
const mockRemove = vi.fn()

vi.mock('~/services/issues.service', () => ({
  issuesService: {
    list: (key: string, filters?: unknown) => mockList(key, filters),
    get: (key: string, id: number) => mockGet(key, id),
    create: (key: string, data: unknown) => mockCreate(key, data),
    update: (key: string, id: number, data: unknown) => mockUpdate(key, id, data),
    remove: (key: string, id: number) => mockRemove(key, id),
  },
}))

const issue = {
  id: 1, key: 'PROJ-1', projectId: 1,
  title: 'Fix bug', description: '', type: 'bug' as const,
  status: 'todo' as const, priority: 'high' as const,
  reporterId: 1,
}

describe('issues store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockList.mockReset()
    mockGet.mockReset()
    mockCreate.mockReset()
    mockUpdate.mockReset()
    mockRemove.mockReset()
  })

  test('create appends issue to list', async () => {
    mockCreate.mockResolvedValueOnce(issue)
    const store = useIssuesStore()
    const result = await store.create('PROJ', { title: 'Fix bug', type: 'bug', priority: 'high' })
    expect(store.issues).toHaveLength(1)
    expect(result.key).toBe('PROJ-1')
  })

  test('update merges issue into list by id', async () => {
    const updated = { ...issue, title: 'Fixed bug' }
    mockUpdate.mockResolvedValueOnce(updated)
    const store = useIssuesStore()
    store.issues = [issue]
    await store.update('PROJ', 1, { title: 'Fixed bug' })
    expect(store.issues[0].title).toBe('Fixed bug')
  })

  test('remove filters deleted issue from list', async () => {
    mockRemove.mockResolvedValueOnce(undefined)
    const store = useIssuesStore()
    store.issues = [issue]
    await store.remove('PROJ', 1)
    expect(store.issues).toHaveLength(0)
  })
})

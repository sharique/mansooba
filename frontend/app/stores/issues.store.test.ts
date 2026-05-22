import { setActivePinia, createPinia } from 'pinia'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { useIssuesStore } from './issues.store'

const mockList = vi.fn()
const mockGet = vi.fn()
const mockCreate = vi.fn()
const mockUpdate = vi.fn()
const mockRemove = vi.fn()
const mockSearch = vi.fn()

vi.mock('~/services/issues.service', () => ({
  issuesService: {
    list: (key: string, filters?: unknown) => mockList(key, filters),
    get: (key: string, id: number) => mockGet(key, id),
    create: (key: string, data: unknown) => mockCreate(key, data),
    update: (key: string, id: number, data: unknown) => mockUpdate(key, id, data),
    remove: (key: string, id: number) => mockRemove(key, id),
    search: (key: string, filters: unknown) => mockSearch(key, filters),
  },
}))

const issue = {
  id: 1, key: 'PROJ-1', project_id: 1,
  title: 'Fix bug', description: '', type: 'bug' as const,
  status: 'todo' as const, priority: 'high' as const,
  reporter_id: 1,
}

describe('issues store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockList.mockReset()
    mockGet.mockReset()
    mockCreate.mockReset()
    mockUpdate.mockReset()
    mockRemove.mockReset()
    mockSearch.mockReset()
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
    const issue0 = store.issues[0]
    expect(issue0).toBeDefined()
    expect(issue0!.title).toBe('Fixed bug')
  })

  test('remove filters deleted issue from list', async () => {
    mockRemove.mockResolvedValueOnce(undefined)
    const store = useIssuesStore()
    store.issues = [issue]
    await store.remove('PROJ', 1)
    expect(store.issues).toHaveLength(0)
  })

  test('searchIssues calls service with filters and updates searchResults', async () => {
    vi.mocked(mockSearch).mockResolvedValue([
      { id: 1, key: 'P-1', title: 'Bug', project_id: 1, type: 'bug',
        status: 'todo', priority: 'high', reporter_id: 1, description: '',
        created_at: '' },
    ])
    const store = useIssuesStore()
    await store.searchIssues('PROJ', { q: 'bug' })
    expect(store.searchResults).toHaveLength(1)
    expect(store.searchResults[0]!.key).toBe('P-1')
    // issues list must not be overwritten
    expect(store.issues).toHaveLength(0)
  })

  test('searchIssues with empty filters clears searchResults without calling service', async () => {
    const store = useIssuesStore()
    store.searchResults = [issue]
    await store.searchIssues('PROJ', {})
    expect(store.searchResults).toHaveLength(0)
    expect(mockSearch).not.toHaveBeenCalled()
  })
})

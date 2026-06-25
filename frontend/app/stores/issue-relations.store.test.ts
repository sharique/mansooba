import { setActivePinia, createPinia } from 'pinia'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { useIssueRelationsStore } from './issue-relations.store'
import type { RelationResponse } from '~/types/domain.types'

const mockList = vi.fn()
const mockCreate = vi.fn()
const mockRemove = vi.fn()

vi.mock('~/services/issue-relations.service', () => ({
  issueRelationsService: {
    list: (id: number) => mockList(id),
    create: (id: number, payload: unknown) => mockCreate(id, payload),
    remove: (id: number, rid: number) => mockRemove(id, rid),
  },
}))

const relation: RelationResponse = {
  id: 1,
  relation_type: 'blocks',
  related_issue: { id: 42, key: 'PROJ-7', title: 'Fix login', status: 'in_progress' },
}

describe('issue-relations store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockList.mockReset()
    mockCreate.mockReset()
    mockRemove.mockReset()
  })

  test('fetchForIssue() populates relations list', async () => {
    mockList.mockResolvedValueOnce([relation])
    const store = useIssueRelationsStore()
    await store.fetchForIssue(1)
    expect(store.relations).toHaveLength(1)
    expect(store.relations[0]!.relation_type).toBe('blocks')
  })

  test('create() appends a new entry', async () => {
    mockCreate.mockResolvedValueOnce(relation)
    const store = useIssueRelationsStore()
    await store.create(1, { target_issue_id: 42, relation_type: 'blocks' })
    expect(store.relations).toHaveLength(1)
    expect(mockCreate).toHaveBeenCalledWith(1, { target_issue_id: 42, relation_type: 'blocks' })
  })

  test('remove() deletes by ID', async () => {
    mockList.mockResolvedValueOnce([relation])
    mockRemove.mockResolvedValueOnce(undefined)
    const store = useIssueRelationsStore()
    await store.fetchForIssue(1)
    await store.remove(1, 1)
    expect(store.relations).toHaveLength(0)
  })
})

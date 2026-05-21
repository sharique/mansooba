import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useCommentsStore } from '~/stores/comments.store'
import * as commentsService from '~/services/comments.service'

vi.mock('~/services/comments.service')

describe('useCommentsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('fetchComments populates comments list', async () => {
    vi.mocked(commentsService.commentsService.list).mockResolvedValue([
      { id: 1, issue_id: 5, author_id: 1, author_name: 'Alice', body: 'hello', created_at: '', updated_at: '' },
    ])
    const store = useCommentsStore()
    await store.fetchComments(5)
    expect(store.comments).toHaveLength(1)
    expect(store.comments[0]?.body).toBe('hello')
  })

  it('addComment appends to list', async () => {
    vi.mocked(commentsService.commentsService.create).mockResolvedValue(
      { id: 2, issue_id: 5, author_id: 1, author_name: 'Alice', body: 'new', created_at: '', updated_at: '' }
    )
    const store = useCommentsStore()
    await store.addComment(5, 'new')
    expect(store.comments).toHaveLength(1)
  })

  it('deleteComment removes from list', async () => {
    vi.mocked(commentsService.commentsService.list).mockResolvedValue([
      { id: 1, issue_id: 5, author_id: 1, author_name: 'Alice', body: 'bye', created_at: '', updated_at: '' },
    ])
    vi.mocked(commentsService.commentsService.delete).mockResolvedValue(undefined)
    const store = useCommentsStore()
    await store.fetchComments(5)
    await store.deleteComment(5, 1)
    expect(store.comments).toHaveLength(0)
  })
})

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useNotificationsStore } from '~/stores/notifications.store'
import * as notificationsService from '~/services/notifications.service'

vi.mock('~/services/notifications.service')

describe('useNotificationsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('fetchUnread populates unread list', async () => {
    vi.mocked(notificationsService.notificationsService.listUnread).mockResolvedValue([
      { id: 1, recipient_id: 1, actor_id: 2, issue_id: 3, issue_key: 'PROJ-3', project_key: 'PROJ', comment_id: 4, read: false, created_at: '' },
    ])
    const store = useNotificationsStore()
    await store.fetchUnread()
    expect(store.unread).toHaveLength(1)
  })

  it('unreadCount reflects list length', async () => {
    vi.mocked(notificationsService.notificationsService.listUnread).mockResolvedValue([
      { id: 1, recipient_id: 1, actor_id: 2, issue_id: 3, issue_key: 'PROJ-3', project_key: 'PROJ', comment_id: 4, read: false, created_at: '' },
      { id: 2, recipient_id: 1, actor_id: 3, issue_id: 5, issue_key: 'PROJ-5', project_key: 'PROJ', comment_id: 6, read: false, created_at: '' },
    ])
    const store = useNotificationsStore()
    await store.fetchUnread()
    expect(store.unreadCount).toBe(2)
  })

  it('markRead removes notification from unread list', async () => {
    vi.mocked(notificationsService.notificationsService.listUnread).mockResolvedValue([
      { id: 1, recipient_id: 1, actor_id: 2, issue_id: 3, issue_key: 'PROJ-3', project_key: 'PROJ', comment_id: 4, read: false, created_at: '' },
    ])
    vi.mocked(notificationsService.notificationsService.markRead).mockResolvedValue(undefined)
    const store = useNotificationsStore()
    await store.fetchUnread()
    await store.markRead(1)
    expect(store.unread).toHaveLength(0)
    expect(store.unreadCount).toBe(0)
  })
})

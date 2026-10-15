// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import NotificationBell from './NotificationBell.vue'
import type { Notification } from '~/types/domain.types'

const push = vi.fn()
const markRead = vi.fn()
const fetchAll = vi.fn()
const notifications = reactive({ unread: [] as Notification[], unreadCount: 0, markRead })
const projects = reactive({ projects: [{ key: 'apol' }] as { key: string }[], fetchAll })

vi.mock('~/stores/notifications.store', () => ({ useNotificationsStore: () => notifications }))
vi.stubGlobal('useProjectsStore', () => projects)
vi.stubGlobal('useRouter', () => ({ push }))
vi.stubGlobal('useTimeFormatter', () => ({ formatDate: () => 'now' }))

const base: Notification = {
  id: 1, recipient_id: 2, actor_id: 3, actor_name: 'Priya', kind: 'mention',
  issue_id: 128, issue_key: 'APOL-12', project_key: 'apol', project_name: 'Apollo',
  comment_id: 0, read: false, created_at: '2026-10-14T09:30:00Z',
}

function show(n: Partial<Notification>) {
  notifications.unread = [{ ...base, ...n }]
  return mount(NotificationBell)
}

describe('NotificationBell text', () => {
  beforeEach(() => {
    push.mockReset()
    markRead.mockReset().mockResolvedValue(undefined)
    fetchAll.mockReset().mockResolvedValue(undefined)
    projects.projects = [{ key: 'apol' }]
  })

  it.each([
    ['mention', 'You were mentioned in APOL-12'],
    ['task_assigned', 'Priya assigned you to APOL-12'],
    ['task_unassigned', 'Priya unassigned you from APOL-12'],
    ['project_member_added', 'Priya added you to Apollo'],
    ['project_member_removed', 'Priya removed you from Apollo'],
  ])('%s reads "%s"', (kind, text) => {
    expect(show({ kind }).text()).toContain(text)
  })

  it('falls back to "Someone" and "a project" when names are missing', () => {
    expect(show({ kind: 'task_assigned', actor_name: '' }).text()).toContain('Someone assigned you to APOL-12')
    expect(show({ kind: 'project_member_added', project_name: '' }).text()).toContain('Priya added you to a project')
  })

  it('renders an unknown kind generically', () => {
    expect(show({ kind: 'something_new' }).text()).toContain('You have a new notification')
  })
})

describe('NotificationBell routing', () => {
  beforeEach(() => {
    push.mockReset()
    markRead.mockReset().mockResolvedValue(undefined)
    fetchAll.mockReset().mockResolvedValue(undefined)
    projects.projects = [{ key: 'apol' }]
  })

  async function click(n: Partial<Notification>) {
    const w = show(n)
    await w.find('li button.text-left').trigger('click')
    await flushPromises()
  }

  it.each(['mention', 'task_assigned', 'task_unassigned'])('%s opens the task and marks it read', async (kind) => {
    await click({ kind })
    expect(markRead).toHaveBeenCalledWith(1)
    expect(push).toHaveBeenCalledWith('/projects/apol/issues/128')
  })

  it('project_member_added opens the project', async () => {
    await click({ kind: 'project_member_added', issue_id: 0, issue_key: '' })
    expect(push).toHaveBeenCalledWith('/projects/apol')
  })

  it('project_member_removed always opens the project list', async () => {
    await click({ kind: 'project_member_removed', issue_id: 0, issue_key: '' })
    expect(push).toHaveBeenCalledWith('/projects')
  })

  it('opens the project list when the project is not in the user\'s list', async () => {
    projects.projects = [{ key: 'other' }]
    await click({ kind: 'task_assigned' })
    expect(push).toHaveBeenCalledWith('/projects')
  })

  it('matches the project key case-insensitively (seeded projects are stored as DEMO)', async () => {
    projects.projects = [{ key: 'demo' }]
    await click({ kind: 'task_assigned', project_key: 'DEMO', issue_key: 'DEMO-3' })
    expect(push).toHaveBeenCalledWith('/projects/demo/issues/128')
  })

  it('loads the projects list before deciding when it is empty', async () => {
    projects.projects = []
    fetchAll.mockImplementation(async () => { projects.projects = [{ key: 'apol' }] })
    await click({ kind: 'task_assigned' })
    expect(fetchAll).toHaveBeenCalled()
    expect(push).toHaveBeenCalledWith('/projects/apol/issues/128')
  })

  it('an unknown kind opens the project when known, else the project list', async () => {
    await click({ kind: 'something_new', issue_id: 0, issue_key: '' })
    expect(push).toHaveBeenLastCalledWith('/projects/apol')
    projects.projects = [{ key: 'other' }]
    await click({ kind: 'something_new', issue_id: 0, issue_key: '' })
    expect(push).toHaveBeenLastCalledWith('/projects')
  })
})

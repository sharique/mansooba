// @vitest-environment happy-dom
import { shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import IssueDetail from '../IssueDetail.vue'
import { stubPermissions, MEMBER, VIEWER, ADMIN } from '../../../../test-helpers/permissions'

const perms = stubPermissions(MEMBER, 'member')

const mockUpdate = vi.fn()
const mockFetchActivity = vi.fn()

vi.stubGlobal('useIssuesStore', () => ({ update: mockUpdate, remove: vi.fn() }))
vi.stubGlobal('useAuthStore', () => ({ user: { id: 1 } }))
vi.stubGlobal('useToast', () => ({ showError: vi.fn(), showSuccess: vi.fn() }))
vi.stubGlobal('useTimeFormatter', () => ({ formatDate: (d: string) => d }))
vi.stubGlobal('useMarkdown', (s: string) => s)
vi.mock('~/stores/comments.store', () => ({
  useCommentsStore: () => ({ comments: [], activity: [], fetchActivity: mockFetchActivity, fetchComments: vi.fn() }),
}))

const issue = {
  id: 7, title: 'Fix it', description: '', type: 'task', priority: 'medium', status: 'todo',
  assignee_id: 3, reporter_id: 1, created_at: '2026-10-01T00:00:00Z',
} as never
const members = [
  { user_id: 3, name: 'Priya', email: 'priya@example.com', role: 'member' },
  { user_id: 4, name: 'Sam', email: 'sam@example.com', role: 'member' },
]

function assigneeSelect(w: ReturnType<typeof shallowMount>) {
  return w.findAll('select')[2]!
}

describe('IssueDetail assignee', () => {
  beforeEach(() => {
    perms.permissions = MEMBER
    perms.myRole = 'member'
    mockUpdate.mockReset().mockResolvedValue({})
    mockFetchActivity.mockReset()
  })

  test('choosing Unassigned sends assignee_id 0, not null', async () => {
    const w = shallowMount(IssueDetail, { props: { issue, projectKey: 'PROJ', members } })
    await assigneeSelect(w).setValue('')
    expect(mockUpdate).toHaveBeenCalledWith('PROJ', 7, { assignee_id: 0 })
  })

  test('choosing a person sends their numeric id', async () => {
    const w = shallowMount(IssueDetail, { props: { issue, projectKey: 'PROJ', members } })
    await assigneeSelect(w).setValue('4')
    expect(mockUpdate).toHaveBeenCalledWith('PROJ', 7, { assignee_id: 4 })
  })
})

describe('IssueDetail roles', () => {
  beforeEach(() => {
    perms.permissions = MEMBER
    perms.myRole = 'member'
    mockUpdate.mockReset().mockResolvedValue({})
  })
  const mountIt = (i: object = issue) => shallowMount(IssueDetail, { props: { issue: i as never, projectKey: 'PROJ', members } })

  test('a viewer gets disabled fields with a reason and no delete button', () => {
    perms.permissions = VIEWER
    perms.myRole = 'viewer'
    const w = mountIt()
    for (const sel of w.findAll('select')) {
      expect(sel.attributes('disabled')).toBeDefined()
      expect(sel.attributes('title')).toBe('denied:task.write')
    }
    expect(w.find('input[type="number"]').attributes('disabled')).toBeDefined()
    expect(w.find('[data-testid="read-only-note"]').exists()).toBe(true)
    expect(w.text()).not.toContain('Delete Issue')
  })

  test('a viewer clicking the title does not start editing', async () => {
    perms.permissions = VIEWER
    perms.myRole = 'viewer'
    const w = mountIt()
    await w.find('h2').trigger('click')
    expect(w.find('input.input-bordered.text-xl').exists()).toBe(false)
  })

  test('a member can edit fields and delete their own task only', () => {
    expect(mountIt().findAll('select')[0]!.attributes('disabled')).toBeUndefined()
    expect(mountIt().text()).toContain('Delete Issue')
    expect(mountIt({ ...(issue as object), reporter_id: 99 }).text()).not.toContain('Delete Issue')
  })

  test('an admin can delete another person\'s task', () => {
    perms.permissions = ADMIN
    perms.myRole = 'admin'
    expect(mountIt({ ...(issue as object), reporter_id: 99 }).text()).toContain('Delete Issue')
  })
})

// @vitest-environment happy-dom
import { shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import IssueDetail from '../IssueDetail.vue'

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

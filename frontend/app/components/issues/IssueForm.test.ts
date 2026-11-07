// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import IssueForm from './IssueForm.vue'
import { stubPermissions, MEMBER, VIEWER } from '../../../test-helpers/permissions'

const perms = stubPermissions(MEMBER, 'member')

const mockCreate = vi.fn()
const mockUpdate = vi.fn()

vi.mock('~/stores/issues.store', () => ({
  useIssuesStore: () => ({ create: mockCreate, update: mockUpdate }),
}))

vi.mock('~/services/projects.service', () => ({
  projectsService: { listMembers: vi.fn().mockResolvedValue([]) },
}))

vi.stubGlobal('useIssuesStore', () => ({ create: mockCreate, update: mockUpdate }))
vi.stubGlobal('useToast', () => ({ showSuccess: vi.fn(), showError: vi.fn() }))
vi.stubGlobal('onMounted', (fn: () => void) => fn())

describe('IssueForm', () => {
  beforeEach(() => {
    perms.permissions = MEMBER
    perms.myRole = 'member'
    setActivePinia(createPinia())
    mockCreate.mockReset()
    mockUpdate.mockReset()
  })

  test('does not emit saved when title is empty', async () => {
    const wrapper = mount(IssueForm, {
      props: { projectKey: 'PROJ' },
    })
    await wrapper.find('[data-testid="submit"]').trigger('click')
    expect(wrapper.emitted('saved')).toBeUndefined()
    expect(mockCreate).not.toHaveBeenCalled()
  })

  const existing = (assignee_id?: number) => ({
    id: 7, title: 'Fix it', description: '', type: 'task', priority: 'medium',
    status: 'todo', assignee_id, story_points: undefined,
  }) as never

  test('edit: choosing Unassigned on an assigned issue sends assignee_id 0', async () => {
    mockUpdate.mockResolvedValueOnce({})
    const wrapper = mount(IssueForm, { props: { projectKey: 'PROJ', issue: existing(3) } })
    ;(wrapper.vm as any).form.assignee_id = undefined
    await wrapper.find('form').trigger('submit')
    await Promise.resolve()
    expect(mockUpdate).toHaveBeenCalledTimes(1)
    expect(mockUpdate.mock.calls[0]![2]).toMatchObject({ assignee_id: 0 })
  })

  test('edit: an issue that had no assignee omits assignee_id', async () => {
    mockUpdate.mockResolvedValueOnce({})
    const wrapper = mount(IssueForm, { props: { projectKey: 'PROJ', issue: existing(undefined) } })
    await wrapper.find('form').trigger('submit')
    await Promise.resolve()
    expect(mockUpdate.mock.calls[0]![2]).not.toHaveProperty('assignee_id', 0)
    expect(mockUpdate.mock.calls[0]![2].assignee_id).toBeUndefined()
  })

  test('edit: keeping an assignee sends their id', async () => {
    mockUpdate.mockResolvedValueOnce({})
    const wrapper = mount(IssueForm, { props: { projectKey: 'PROJ', issue: existing(3) } })
    await wrapper.find('form').trigger('submit')
    await Promise.resolve()
    expect(mockUpdate.mock.calls[0]![2].assignee_id).toBe(3)
  })

  test('create: never sends assignee_id 0', async () => {
    mockCreate.mockResolvedValueOnce({})
    const wrapper = mount(IssueForm, { props: { projectKey: 'PROJ' } })
    await wrapper.find('[data-testid="title"]').setValue('New')
    await wrapper.find('form').trigger('submit')
    await Promise.resolve()
    expect(mockCreate.mock.calls[0]![1].assignee_id).toBeUndefined()
  })
})

describe('IssueForm roles', () => {
  beforeEach(() => {
    perms.permissions = MEMBER
    perms.myRole = 'member'
    setActivePinia(createPinia())
    mockCreate.mockReset()
  })

  test('a viewer cannot submit and sees why', async () => {
    perms.permissions = VIEWER
    perms.myRole = 'viewer'
    const wrapper = mount(IssueForm, { props: { projectKey: 'PROJ' } })
    await wrapper.find('[data-testid="title"]').setValue('Sneaky')
    const submit = wrapper.find('[data-testid="submit"]')
    expect(submit.attributes('disabled')).toBeDefined()
    expect(wrapper.find('[data-testid="read-only-note"]').text()).toBe('denied:task.write')
    await wrapper.find('form').trigger('submit')
    expect(mockCreate).not.toHaveBeenCalled()
    perms.permissions = MEMBER
    perms.myRole = 'member'
  })

  test('a member can submit', async () => {
    mockCreate.mockResolvedValueOnce({ id: 1 })
    const wrapper = mount(IssueForm, { props: { projectKey: 'PROJ' } })
    await wrapper.find('[data-testid="title"]').setValue('Real work')
    expect(wrapper.find('[data-testid="read-only-note"]').exists()).toBe(false)
    await wrapper.find('form').trigger('submit')
    expect(mockCreate).toHaveBeenCalled()
  })

  describe('sprint when creating', () => {
    const sprints = [
      { id: '1', name: 'Sprint 1', status: 'completed' },
      { id: '2', name: 'Sprint 2', status: 'active' },
      { id: '3', name: 'Sprint 3', status: 'planning' },
    ] as never[]

    const options = (w: ReturnType<typeof mount>) =>
      w.findAll('[data-testid="sprint"] option').map(o => o.text())

    test('offers the backlog first, then only the current and future sprints', () => {
      const w = mount(IssueForm, { props: { projectKey: 'PROJ', sprints } })
      const list = options(w)
      expect(list[0]).toBe('Backlog (no sprint)')
      expect(list).toHaveLength(3)
      expect(list.join(' ')).toContain('Sprint 2')
      expect(list.join(' ')).toContain('Sprint 3')
      expect(list.join(' ')).not.toContain('Sprint 1')
      expect((w.find('[data-testid="sprint"]').element as HTMLSelectElement).value).toBe('')
    })

    test('says which sprint is the current one', () => {
      const w = mount(IssueForm, { props: { projectKey: 'PROJ', sprints } })
      expect(options(w).find(o => o.includes('Sprint 2'))).toContain('active')
      expect(options(w).find(o => o.includes('Sprint 3'))).toContain('planned')
    })

    test('by default the new issue goes to the backlog: no sprint is sent', async () => {
      mockCreate.mockResolvedValueOnce({})
      const w = mount(IssueForm, { props: { projectKey: 'PROJ', sprints } })
      await w.find('[data-testid="title"]').setValue('A task')
      await w.find('form').trigger('submit')
      await Promise.resolve()
      expect(mockCreate).toHaveBeenCalledTimes(1)
      expect(mockCreate.mock.calls[0]![1].sprint_id).toBeUndefined()
    })

    test('a chosen sprint is sent as a number', async () => {
      mockCreate.mockResolvedValueOnce({})
      const w = mount(IssueForm, { props: { projectKey: 'PROJ', sprints } })
      await w.find('[data-testid="title"]').setValue('A task')
      await w.find('[data-testid="sprint"]').setValue('3')
      await w.find('form').trigger('submit')
      await Promise.resolve()
      expect(mockCreate.mock.calls[0]![1].sprint_id).toBe(3)
    })

    test('is not shown without sprints, nor when only finished ones exist', () => {
      expect(mount(IssueForm, { props: { projectKey: 'PROJ' } }).find('[data-testid="sprint"]').exists()).toBe(false)
      const onlyDone = [{ id: '1', name: 'Old', status: 'completed' }] as never[]
      expect(mount(IssueForm, { props: { projectKey: 'PROJ', sprints: onlyDone } }).find('[data-testid="sprint"]').exists()).toBe(false)
    })

    test('is not shown when editing (moving a task between sprints is done elsewhere)', () => {
      const issue = { id: 7, title: 'Fix it', description: '', type: 'task', priority: 'medium', status: 'todo' } as never
      const w = mount(IssueForm, { props: { projectKey: 'PROJ', sprints, issue } })
      expect(w.find('[data-testid="sprint"]').exists()).toBe(false)
    })
  })
})

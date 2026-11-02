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
})

// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, expect, test, vi } from 'vitest'
import BacklogIssueRow from './BacklogIssueRow.vue'
import type { Issue, Sprint } from '~/types/domain.types'

vi.stubGlobal('navigateTo', vi.fn())

const makeIssue = (overrides: Partial<Issue> = {}): Issue => ({
  id: 1,
  key: 'PROJ-1',
  projectId: 1,
  title: 'Issue 1',
  description: '',
  type: 'task',
  status: 'backlog',
  priority: 'medium',
  reporterId: 1,
  ...overrides,
})

const makeSprint = (id: number, name: string, status: Sprint['status'] = 'Planning'): Sprint => ({
  id: String(id),
  project_id: '1',
  name,
  goal: '',
  status,
  start_date: null,
  end_date: null,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
})

describe('BacklogIssueRow sprint assignment', () => {
  test('shows sprint select when canManage=true and sprints are available', () => {
    const sprints = [makeSprint(1, 'Sprint 1'), makeSprint(2, 'Sprint 2')]
    const wrapper = mount(BacklogIssueRow, {
      props: { issue: makeIssue(), projectKey: 'PROJ', canManage: true, sprints },
    })
    expect(wrapper.find('select').exists()).toBe(true)
  })

  test('hides sprint select when canManage=false', () => {
    const sprints = [makeSprint(1, 'Sprint 1')]
    const wrapper = mount(BacklogIssueRow, {
      props: { issue: makeIssue(), projectKey: 'PROJ', canManage: false, sprints },
    })
    expect(wrapper.find('select').exists()).toBe(false)
  })

  test('hides sprint select when no sprints available', () => {
    const wrapper = mount(BacklogIssueRow, {
      props: { issue: makeIssue(), projectKey: 'PROJ', canManage: true, sprints: [] },
    })
    expect(wrapper.find('select').exists()).toBe(false)
  })

  test('emits sprint-assign with issueId and sprintId when sprint is selected', async () => {
    const sprints = [makeSprint(7, 'Sprint 7')]
    const wrapper = mount(BacklogIssueRow, {
      props: { issue: makeIssue({ id: 42 }), projectKey: 'PROJ', canManage: true, sprints },
    })
    const select = wrapper.find('select')
    await select.setValue('7')
    expect(wrapper.emitted('sprint-assign')).toBeTruthy()
    expect(wrapper.emitted('sprint-assign')![0]).toEqual([{ issueId: 42, sprintId: 7 }])
  })
})

// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, expect, test, vi } from 'vitest'
import BacklogIssueRow from './BacklogIssueRow.vue'
import type { Issue, Sprint } from '~/types/domain.types'

vi.stubGlobal('navigateTo', vi.fn())

const makeIssue = (overrides: Partial<Issue> = {}): Issue => ({
  id: 1,
  key: 'PROJ-1',
  project_id: 1,
  title: 'Issue 1',
  description: '',
  type: 'task',
  status: 'backlog',
  priority: 'medium',
  reporter_id: 1,
  ...overrides,
})

const makeSprint = (id: number, name: string, status: Sprint['status'] = 'planning'): Sprint => ({
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
  test('shows the sprint select to any member when a sprint is open', () => {
    const sprints = [makeSprint(1, 'Sprint 1'), makeSprint(2, 'Sprint 2')]
    const wrapper = mount(BacklogIssueRow, {
      props: { issue: makeIssue(), projectKey: 'PROJ', sprints },
    })
    expect(wrapper.find('select').exists()).toBe(true)
  })

  test('hides sprint select when every sprint is completed', () => {
    const sprints = [makeSprint(1, 'Sprint 1', 'completed')]
    const wrapper = mount(BacklogIssueRow, {
      props: { issue: makeIssue(), projectKey: 'PROJ', sprints },
    })
    expect(wrapper.find('select').exists()).toBe(false)
  })

  test('hides sprint select when no sprints available', () => {
    const wrapper = mount(BacklogIssueRow, {
      props: { issue: makeIssue(), projectKey: 'PROJ', sprints: [] },
    })
    expect(wrapper.find('select').exists()).toBe(false)
  })

  test('emits sprint-assign with issueId and sprintId when sprint is selected', async () => {
    const sprints = [makeSprint(7, 'Sprint 7')]
    const wrapper = mount(BacklogIssueRow, {
      props: { issue: makeIssue({ id: 42 }), projectKey: 'PROJ', sprints },
    })
    const select = wrapper.find('select')
    await select.setValue('7')
    expect(wrapper.emitted('sprint-assign')).toBeTruthy()
    expect(wrapper.emitted('sprint-assign')![0]).toEqual([{ issueId: 42, sprintId: 7 }])
  })
})

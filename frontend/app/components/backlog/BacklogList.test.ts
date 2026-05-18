// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, expect, test } from 'vitest'
import BacklogList from './BacklogList.vue'
import type { Issue } from '~/types/domain.types'

const makeIssue = (id: number, overrides: Partial<Issue> = {}): Issue => ({
  id,
  key: `PROJ-${id}`,
  project_id: 1,
  title: `Issue ${id}`,
  description: '',
  type: 'task',
  status: 'backlog',
  priority: 'medium',
  reporter_id: 1,
  ...overrides,
})

describe('BacklogList', () => {
  test('renders a stub row for each issue', () => {
    const issues = [makeIssue(1, { title: 'Issue A' }), makeIssue(2, { title: 'Issue B' })]
    const wrapper = mount(BacklogList, {
      props: { issues, projectKey: 'PROJ' },
      global: { stubs: { BacklogIssueRow: true } },
    })
    expect(wrapper.findAllComponents({ name: 'BacklogIssueRow' })).toHaveLength(2)
  })

  test('shows empty state when issues list is empty', () => {
    const wrapper = mount(BacklogList, {
      props: { issues: [], projectKey: 'PROJ' },
    })
    expect(wrapper.text()).toContain('No issues in the backlog')
  })

  test('shows 5 loading skeletons when loading prop is true', () => {
    const wrapper = mount(BacklogList, {
      props: { issues: [], projectKey: 'PROJ', loading: true },
    })
    expect(wrapper.findAll('.skeleton')).toHaveLength(5)
  })

  test('does not show empty state while loading', () => {
    const wrapper = mount(BacklogList, {
      props: { issues: [], projectKey: 'PROJ', loading: true },
    })
    expect(wrapper.text()).not.toContain('No issues in the backlog')
  })
})

// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, expect, test, vi } from 'vitest'
import BoardCard from './BoardCard.vue'

vi.stubGlobal('navigateTo', vi.fn())

const issue = {
  id: 1, key: 'PROJ-1', project_id: 1,
  title: 'Fix the bug', description: '',
  type: 'bug' as const, status: 'todo' as const,
  priority: 'critical' as const, reporter_id: 1,
}

describe('BoardCard', () => {
  test('displays issue key and title', () => {
    const wrapper = mount(BoardCard, { props: { issue, projectKey: 'PROJ' } })
    expect(wrapper.text()).toContain('PROJ-1')
    expect(wrapper.text()).toContain('Fix the bug')
  })

  test('priority badge has badge-error class for critical', () => {
    const wrapper = mount(BoardCard, { props: { issue, projectKey: 'PROJ' } })
    expect(wrapper.find('.badge').classes()).toContain('badge-error')
  })

  test('card has priority border class for critical', () => {
    const wrapper = mount(BoardCard, { props: { issue, projectKey: 'PROJ' } })
    expect(wrapper.find('.card').classes()).toContain('border-l-error')
  })

  test('card does not render a status select', () => {
    const wrapper = mount(BoardCard, { props: { issue, projectKey: 'PROJ' } })
    expect(wrapper.find('select').exists()).toBe(false)
  })

})

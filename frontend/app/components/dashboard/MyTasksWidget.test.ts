// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '~/stores/auth.store'
import MyTasksWidget from './MyTasksWidget.vue'
import type { Issue } from '~/types/domain.types'

// NuxtLink must render its slot so inner spans are accessible in tests
const NuxtLinkStub = { template: '<a><slot /></a>' }

function makeIssue(overrides: Partial<Issue>): Issue {
  return {
    id: 1,
    key: 'PROJ-1',
    title: 'Test issue',
    status: 'todo',
    priority: 'medium',
    project_id: 1,
    reporter_id: 1,
    sprint_id: null,
    created_at: '',
    updated_at: '',
    ...overrides,
  }
}

const stubs = { NuxtLink: NuxtLinkStub, Icon: true, UiEmptyState: true }

describe('MyTasksWidget', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('shows skeleton while loading', () => {
    const w = mount(MyTasksWidget, { props: { loading: true }, global: { stubs } })
    expect(w.find('.skeleton').exists()).toBe(true)
  })

  it('shows EmptyState when no issues', () => {
    const store = useAuthStore()
    store.myIssues = []
    const w = mount(MyTasksWidget, { props: { loading: false }, global: { stubs } })
    expect(w.findComponent({ name: 'UiEmptyState' }).exists()).toBe(true)
  })

  it('renders card title as "In your tray"', () => {
    const store = useAuthStore()
    store.myIssues = []
    const w = mount(MyTasksWidget, { props: { loading: false }, global: { stubs } })
    expect(w.text()).toContain('In your tray')
  })

  it('sorts in_progress issues before todo and done', () => {
    const store = useAuthStore()
    store.myIssues = [
      makeIssue({ id: 1, key: 'P-1', status: 'done' }),
      makeIssue({ id: 2, key: 'P-2', status: 'in_progress' }),
      makeIssue({ id: 3, key: 'P-3', status: 'todo' }),
    ]
    const w = mount(MyTasksWidget, { props: { loading: false }, global: { stubs } })
    const keys = w.findAll('.font-mono').map(el => el.text())
    expect(keys[0]).toBe('P-2') // in_progress first
    expect(keys[1]).toBe('P-3') // todo second
    expect(keys[2]).toBe('P-1') // done last
  })

  it('caps the list at 10 issues', () => {
    const store = useAuthStore()
    store.myIssues = Array.from({ length: 15 }, (_, i) =>
      makeIssue({ id: i + 1, key: `P-${i + 1}`, status: 'todo' }),
    )
    const w = mount(MyTasksWidget, { props: { loading: false }, global: { stubs } })
    expect(w.findAll('.font-mono').length).toBe(10)
  })
})

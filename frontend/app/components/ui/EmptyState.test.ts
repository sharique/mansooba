// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, it, expect } from 'vitest'
import EmptyState from './EmptyState.vue'

const stubs = { Icon: true }

describe('EmptyState', () => {
  it('renders title and description', () => {
    const w = mount(EmptyState, {
      props: { title: 'No issues', description: 'Create one to get started' },
      global: { stubs },
    })
    expect(w.text()).toContain('No issues')
    expect(w.text()).toContain('Create one to get started')
  })

  it('renders the action slot', () => {
    const w = mount(EmptyState, {
      props: { title: 'Empty' },
      slots: { action: '<button class="cta">Add</button>' },
      global: { stubs },
    })
    expect(w.find('.cta').exists()).toBe(true)
  })
})

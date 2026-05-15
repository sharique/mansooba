// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, expect, test } from 'vitest'
import BoardColumn from './BoardColumn.vue'

const makeIssue = (id: number) => ({
  id, key: `PROJ-${id}`, projectId: 1,
  title: `Issue ${id}`, description: '',
  type: 'task' as const, status: 'todo' as const,
  priority: 'low' as const, reporterId: 1,
})

describe('BoardColumn', () => {
  test('renders correct number of BoardCard stubs', () => {
    const column = { status: 'todo', issues: [makeIssue(1), makeIssue(2)] }
    const wrapper = mount(BoardColumn, {
      props: { column, projectKey: 'PROJ' },
      global: { stubs: { BoardCard: true } },
    })
    expect(wrapper.findAllComponents({ name: 'BoardCard' })).toHaveLength(2)
  })

  test('clicking + Add Issue emits createIssue with column status', async () => {
    const column = { status: 'in_progress', issues: [] }
    const wrapper = mount(BoardColumn, {
      props: { column, projectKey: 'PROJ' },
      global: { stubs: { BoardCard: true } },
    })
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('createIssue')).toBeTruthy()
    expect(wrapper.emitted('createIssue')![0]).toEqual(['in_progress'])
  })
})

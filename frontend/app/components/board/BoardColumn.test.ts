// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, expect, test, vi } from 'vitest'
import { VueDraggable } from 'vue-draggable-plus'
import BoardColumn from './BoardColumn.vue'
import { wasJustDragged } from '~/utils/dragGuard'

const makeIssue = (id: number) => ({
  id, key: `PROJ-${id}`, project_id: 1,
  title: `Issue ${id}`, description: '',
  type: 'task' as const, status: 'todo' as const,
  priority: 'low' as const, reporter_id: 1,
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

// happy-dom cannot perform a real pointer drag, so these tests hand the drag list the
// events it would emit. The real drag is checked by the manual walk-through.
describe('BoardColumn drag and drop', () => {
  const mountColumn = (issues = [makeIssue(1)]) =>
    mount(BoardColumn, {
      props: { column: { status: 'in_progress', issues }, projectKey: 'PROJ' },
      global: { stubs: { BoardCard: true } },
    })

  const dropEvent = (dropEffect: string | undefined, children: unknown[] = []) => ({
    item: { dataset: { issueId: '7' } },
    from: { insertBefore: vi.fn(), children },
    oldIndex: 0,
    originalEvent: dropEffect === undefined ? {} : { dataTransfer: { dropEffect } },
  })

  test('keeps a drop area as tall as the empty-column minimum, even with no cards', () => {
    const wrapper = mountColumn([])
    expect(wrapper.find('.min-h-32').exists()).toBe(true)
  })

  test('emits moved with the card id and this column\'s status when a card is dropped on it', () => {
    const wrapper = mountColumn()
    wrapper.findComponent(VueDraggable).vm.$emit('add', dropEvent('move'))
    expect(wrapper.emitted('moved')).toEqual([[{ issueId: 7, toStatus: 'in_progress' }]])
  })

  test('a touch drop (no dataTransfer) is a normal drop', () => {
    const wrapper = mountColumn()
    wrapper.findComponent(VueDraggable).vm.$emit('add', dropEvent(undefined))
    expect(wrapper.emitted('moved')).toHaveLength(1)
  })

  test('a cancelled drag (dropEffect none) moves nothing and puts the card back where it was', () => {
    const wrapper = mountColumn()
    const evt = dropEvent('none')
    wrapper.findComponent(VueDraggable).vm.$emit('add', evt)
    expect(wrapper.emitted('moved')).toBeUndefined()
    expect(evt.from.insertBefore).toHaveBeenCalledWith(evt.item, null)
  })

  test('puts a cancelled card back before the card that now sits at its old index', () => {
    const wrapper = mountColumn()
    const neighbour = { id: 'neighbour' }
    const evt = dropEvent('none', [neighbour])
    wrapper.findComponent(VueDraggable).vm.$emit('add', evt)
    expect(evt.from.insertBefore).toHaveBeenCalledWith(evt.item, neighbour)
  })

  test('marks the end of a drag so the click that follows does not open the task', () => {
    const wrapper = mountColumn()
    wrapper.findComponent(VueDraggable).vm.$emit('end', {})
    expect(wasJustDragged()).toBe(true)
  })

  test('highlights itself while a card is dragged over it', () => {
    const wrapper = mountColumn()
    expect(wrapper.find('.card').classes().join(' ')).toContain('has-[.sortable-ghost]:ring-2')
  })

  test('stops locked cards being picked up but still lets them be clicked', () => {
    const list = mountColumn().findComponent(VueDraggable)
    expect(list.props('filter')).toBe('.no-drag')
    expect(list.props('preventOnFilter')).toBe(false)
  })

  test('forwards a move chosen from a card\'s keyboard control, exactly like a drop', () => {
    const wrapper = mountColumn()
    wrapper.findComponent({ name: 'BoardCard' }).vm.$emit('move', { issueId: 1, toStatus: 'done' })
    expect(wrapper.emitted('moved')).toEqual([[{ issueId: 1, toStatus: 'done' }]])
  })
})

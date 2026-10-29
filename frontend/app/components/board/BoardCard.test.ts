// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import BoardCard from './BoardCard.vue'
import { markDragEnd } from '~/utils/dragGuard'
import type { Issue } from '~/types/domain.types'

const navigate = vi.fn()
vi.stubGlobal('navigateTo', navigate)

const makeIssue = (overrides: Partial<Issue> = {}): Issue => ({
  id: 3, key: 'PROJ-3', project_id: 1, title: 'Fix it', description: '',
  type: 'task', status: 'todo', priority: 'high', reporter_id: 1, attachment_count: 0,
  ...overrides,
})

const mountCard = (overrides: Partial<Issue> = {}) =>
  mount(BoardCard, {
    props: { issue: makeIssue(overrides), projectKey: 'PROJ' },
    global: { stubs: { UserAvatar: true } },
  })

describe('BoardCard', () => {
  beforeEach(() => {
    navigate.mockClear()
    markDragEnd(0)
  })

  test('carries its id so the drag list can tell which card was dropped', () => {
    expect(mountCard().attributes('data-issue-id')).toBe('3')
  })

  test('clicking the card opens the task', async () => {
    await mountCard().trigger('click')
    expect(navigate).toHaveBeenCalledWith('/projects/PROJ/issues/3')
  })

  test('a click straight after a drag ends does not open the task', async () => {
    markDragEnd()
    await mountCard().trigger('click')
    expect(navigate).not.toHaveBeenCalled()
  })

  test('a click well after a drag opens the task again', async () => {
    markDragEnd(Date.now() - 1000)
    await mountCard().trigger('click')
    expect(navigate).toHaveBeenCalledTimes(1)
  })

  describe('locked card (its sprint is completed)', () => {
    test('is not offered for dragging', () => {
      expect(mountCard({ locked: true }).classes()).toContain('no-drag')
      expect(mountCard({ locked: false }).classes()).not.toContain('no-drag')
    })

    test('says why, with a lock', () => {
      const wrapper = mountCard({ locked: true })
      expect(wrapper.text()).toContain('Sprint completed')
      expect(wrapper.attributes('aria-disabled')).toBe('true')
      expect(wrapper.find('[title="Sprint completed"]').exists()).toBe(true)
    })

    test('an unlocked card shows no lock and is not marked disabled', () => {
      const wrapper = mountCard({ locked: false })
      expect(wrapper.text()).not.toContain('Sprint completed')
      expect(wrapper.attributes('aria-disabled')).toBeUndefined()
    })

    test('can still be opened with a click', async () => {
      await mountCard({ locked: true }).trigger('click')
      expect(navigate).toHaveBeenCalledWith('/projects/PROJ/issues/3')
    })
  })

  describe('"Move to…" control (keyboard and screen-reader alternative to dragging)', () => {
    const mountAttached = (overrides: Partial<Issue> = {}) =>
      mount(BoardCard, {
        props: { issue: makeIssue(overrides), projectKey: 'PROJ' },
        global: { stubs: { UserAvatar: true } },
        attachTo: document.body,
      })
    const toggle = (w: ReturnType<typeof mountAttached>) => w.find('button[aria-label^="Move PROJ-3"]')

    test('is a button named after the task, closed to start with', () => {
      const w = mountAttached()
      expect(toggle(w).exists()).toBe(true)
      expect(toggle(w).attributes('aria-label')).toBe('Move PROJ-3 to…')
      expect(toggle(w).attributes('aria-expanded')).toBe('false')
      expect(w.find('ul').exists()).toBe(false)
      w.unmount()
    })

    test('opens a list of the other columns by their names', async () => {
      const w = mountAttached({ status: 'todo' })
      await toggle(w).trigger('click')
      expect(toggle(w).attributes('aria-expanded')).toBe('true')
      expect(toggle(w).attributes('aria-controls')).toBe(w.find('ul').attributes('id'))
      expect(w.findAll('ul button').map(b => b.text())).toEqual(['In Progress', 'In Review', 'Done'])
      w.unmount()
    })

    test('choosing a column emits move and closes the list', async () => {
      const w = mountAttached({ status: 'todo' })
      await toggle(w).trigger('click')
      await w.findAll('ul button')[2]!.trigger('click')
      expect(w.emitted('move')).toEqual([[{ issueId: 3, toStatus: 'done' }]])
      expect(w.find('ul').exists()).toBe(false)
      expect(toggle(w).attributes('aria-expanded')).toBe('false')
      w.unmount()
    })

    test('Escape closes the list and puts focus back on the button', async () => {
      const w = mountAttached()
      await toggle(w).trigger('click')
      await w.findAll('ul button')[0]!.trigger('keydown', { key: 'Escape' })
      expect(w.find('ul').exists()).toBe(false)
      expect(document.activeElement).toBe(toggle(w).element)
      w.unmount()
    })

    test('Escape also closes the list while the focus is still on the button', async () => {
      const w = mountAttached()
      await toggle(w).trigger('click')
      await toggle(w).trigger('keydown', { key: 'Escape' })
      expect(w.find('ul').exists()).toBe(false)
      w.unmount()
    })

    test('using it does not open the task', async () => {
      const w = mountAttached()
      await toggle(w).trigger('click')
      await w.findAll('ul button')[0]!.trigger('click')
      expect(navigate).not.toHaveBeenCalled()
      w.unmount()
    })

    test('is switched off on a locked card and says why', async () => {
      const w = mountAttached({ locked: true })
      expect(toggle(w).attributes('aria-disabled')).toBe('true')
      expect(toggle(w).attributes('title')).toBe('Sprint completed')
      await toggle(w).trigger('click')
      expect(w.find('ul').exists()).toBe(false)
      expect(w.emitted('move')).toBeUndefined()
      w.unmount()
    })
  })
})

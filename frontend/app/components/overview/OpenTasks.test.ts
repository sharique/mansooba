// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import OpenTasks from './OpenTasks.vue'
import { NuxtLinkStub } from './stubs'
import type { OpenTask, OverviewList } from '~/types/domain.types'

const task = (n: number, extra: Partial<OpenTask> = {}): OpenTask => ({
  issue_id: n, key: `SCR-${n}`, title: `Task ${n}`, status: 'in_progress', priority: 'high',
  assignee: { id: 7, name: 'Priya Nair' }, ...extra,
})
const list = (items: OpenTask[], sprint_active = true): OverviewList<OpenTask> => ({ sprint_active, items })

const mountIt = (props: Record<string, unknown>) =>
  mount(OpenTasks, {
    props: { projectKey: 'scr', data: null, loading: false, error: null, ...props },
    global: { stubs: { NuxtLink: NuxtLinkStub, UserAvatar: true } },
  })

describe('OpenTasks', () => {
  it('shows key, title, status, priority and assignee, each row linking to the task', () => {
    const w = mountIt({ data: list([task(1), task(2, { status: 'in_review', priority: 'critical' })]) })
    const rows = w.findAll('[data-task]')
    expect(rows).toHaveLength(2)
    expect(rows[0]!.text()).toContain('SCR-1')
    expect(rows[0]!.text()).toContain('Task 1')
    expect(rows[0]!.text()).toContain('In Progress')
    expect(rows[0]!.text()).toContain('high')
    expect(rows[0]!.text()).toContain('Priya Nair')
    expect(rows[1]!.text()).toContain('In Review')
    expect(rows[0]!.find('a').attributes('href')).toBe('/projects/scr/issues/1')
  })

  it('keeps the order it is given', () => {
    const w = mountIt({ data: list([task(5), task(2), task(9)]) })
    expect(w.findAll('[data-task]').map(r => r.attributes('data-task'))).toEqual(['SCR-5', 'SCR-2', 'SCR-9'])
  })

  it('shows a placeholder for an unassigned task', () => {
    const w = mountIt({ data: list([task(1, { assignee: null })]) })
    expect(w.find('[data-task]').text()).toMatch(/unassigned/i)
  })

  it('shows "Goto board" whenever a sprint is active, even with few tasks', () => {
    const w = mountIt({ data: list([task(1)]) })
    const link = w.find('a[href="/projects/scr/board"]')
    expect(link.exists()).toBe(true)
    expect(link.text()).toBe('Goto board')
  })

  it('says there are no open tasks when the active sprint has none, and still offers the board', () => {
    const w = mountIt({ data: list([]) })
    expect(w.text()).toMatch(/no open tasks in this sprint/i)
    expect(w.find('a[href="/projects/scr/board"]').exists()).toBe(true)
  })

  it('shows the no-active-sprint message and no board link without a sprint', () => {
    const w = mountIt({ data: list([], false) })
    expect(w.text()).toMatch(/no active sprint/i)
    expect(w.find('a[href="/projects/scr/board"]').exists()).toBe(false)
  })

  it('shows a skeleton while loading', () => {
    expect(mountIt({ loading: true }).find('.skeleton').exists()).toBe(true)
  })

  it('shows the error with a Retry button that emits retry', async () => {
    const w = mountIt({ error: 'nope' })
    expect(w.text()).toContain('nope')
    await w.find('button').trigger('click')
    expect(w.emitted('retry')).toHaveLength(1)
  })
})

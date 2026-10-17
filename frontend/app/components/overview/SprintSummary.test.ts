// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import SprintSummary from './SprintSummary.vue'
import { NuxtLinkStub } from './stubs'
import type { SprintSummary as Summary } from '~/types/domain.types'

vi.stubGlobal('useTimeFormatter', () => ({
  formatDate: (iso: string | null | undefined) => iso ?? '—',
  formatDateTime: (iso: string) => iso,
}))

const sprint: Summary = {
  id: 3, name: 'Sprint 3', start_date: '2026-10-12T00:00:00Z', end_date: '2026-10-26T00:00:00Z', is_overdue: false,
  todo: { tasks: 2, points: 3 }, in_progress: { tasks: 1, points: 5 },
  in_review: { tasks: 4, points: 0 }, done: { tasks: 6, points: 13 }, total: { tasks: 13, points: 21 },
}

const mountIt = (props: Record<string, unknown>) =>
  mount(SprintSummary, {
    props: { projectKey: 'scr', data: null, loading: false, error: null, ...props },
    global: { stubs: { NuxtLink: NuxtLinkStub } },
  })

describe('SprintSummary', () => {
  it('shows the name, both dates and the four counters with tasks and points', () => {
    const w = mountIt({ data: sprint })
    expect(w.text()).toContain('Sprint 3')
    expect(w.text()).toContain('2026-10-12T00:00:00Z')
    expect(w.text()).toContain('2026-10-26T00:00:00Z')
    const counters = w.findAll('[data-counter]')
    expect(counters.map(c => c.attributes('data-counter'))).toEqual(['todo', 'in_progress', 'in_review', 'done'])
    expect(counters[0]!.text()).toMatch(/To Do/i)
    expect(counters[0]!.text()).toContain('2')
    expect(counters[0]!.text()).toContain('3 pts')
    expect(counters[2]!.text()).toMatch(/In Review/i)
    expect(counters[2]!.text()).toContain('4')
    expect(counters[3]!.text()).toContain('13 pts')
  })

  it('shows zero progress for a sprint without tasks instead of an error', () => {
    const zero = { ...sprint, todo: { tasks: 0, points: 0 }, in_progress: { tasks: 0, points: 0 }, in_review: { tasks: 0, points: 0 }, done: { tasks: 0, points: 0 }, total: { tasks: 0, points: 0 } }
    const w = mountIt({ data: zero })
    expect(w.text()).toContain('Sprint 3')
    expect(w.text()).toMatch(/no tasks yet/i)
    expect(w.findAll('[data-counter]')).toHaveLength(4)
  })

  it('flags an overdue sprint', () => {
    expect(mountIt({ data: sprint }).text()).not.toMatch(/overdue/i)
    expect(mountIt({ data: { ...sprint, is_overdue: true } }).text()).toMatch(/overdue/i)
  })

  it('says there is no active sprint and links to Sprints, without zeroed numbers', () => {
    const w = mountIt({ data: null })
    expect(w.text()).toMatch(/no active sprint/i)
    expect(w.find('a[href="/projects/scr/sprints"]').exists()).toBe(true)
    expect(w.findAll('[data-counter]')).toHaveLength(0)
  })

  it('links to the Sprints page when a sprint is shown', () => {
    expect(mountIt({ data: sprint }).find('a[href="/projects/scr/sprints"]').exists()).toBe(true)
  })

  it('shows a skeleton while loading and nothing else', () => {
    const w = mountIt({ loading: true })
    expect(w.find('.skeleton').exists()).toBe(true)
    expect(w.text()).not.toMatch(/no active sprint/i)
  })

  it('shows the error with a Retry button that emits retry', async () => {
    const w = mountIt({ error: 'boom' })
    expect(w.text()).toContain('boom')
    await w.find('button').trigger('click')
    expect(w.emitted('retry')).toHaveLength(1)
  })
})

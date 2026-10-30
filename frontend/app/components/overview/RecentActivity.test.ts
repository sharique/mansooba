// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import RecentActivity from './RecentActivity.vue'
import { NuxtLinkStub } from './stubs'
import type { OverviewList, RecentActivityItem } from '~/types/domain.types'

vi.stubGlobal('useTimeFormatter', () => ({
  formatDate: (iso: string | null | undefined) => iso ?? '—',
  formatDateTime: (iso: string) => `at ${iso}`,
}))

const move = (id: number, to = 'in_review'): RecentActivityItem => ({
  id, kind: 'status_changed', actor: { id: 7, name: 'Priya Nair' }, issue_id: 12, issue_key: 'SCR-12',
  issue_title: 'Fix login', new_status: to, created_at: '2026-10-17T10:00:00Z',
})
const comment = (id: number): RecentActivityItem => ({
  id, kind: 'comment_added', actor: { id: 8, name: 'Sam Smith' }, issue_id: 13, issue_key: 'SCR-13',
  issue_title: 'Add tests', created_at: '2026-10-17T09:00:00Z',
})
const deleted = (id: number): RecentActivityItem => ({
  id, kind: 'issue_deleted', actor: { id: 7, name: 'Priya Nair' }, issue_id: 40, issue_key: 'SCR-40',
  issue_title: 'Old idea', created_at: '2026-10-17T11:00:00Z',
})
const list = (items: RecentActivityItem[], sprint_active = true): OverviewList<RecentActivityItem> => ({ sprint_active, items })

const mountIt = (props: Record<string, unknown>) =>
  mount(RecentActivity, {
    props: { projectKey: 'scr', data: null, loading: false, error: null, scope: 'all', ...props },
    global: { stubs: { NuxtLink: NuxtLinkStub, UserAvatar: true } },
    attachTo: document.body,
  })

describe('RecentActivity', () => {
  it('has two tabs with "All activity" selected first', () => {
    const w = mountIt({ data: list([]) })
    expect(w.find('[role="tablist"]').exists()).toBe(true)
    const tabs = w.findAll('[role="tab"]')
    expect(tabs.map(t => t.text())).toEqual(['All activity', 'Current sprint'])
    expect(tabs[0]!.attributes('aria-selected')).toBe('true')
    expect(tabs[1]!.attributes('aria-selected')).toBe('false')
    expect(w.find('[role="tabpanel"]').exists()).toBe(true)
  })

  it('reflects the scope it is given', () => {
    const w = mountIt({ data: list([]), scope: 'sprint' })
    expect(w.findAll('[role="tab"]')[1]!.attributes('aria-selected')).toBe('true')
  })

  it('selecting a tab emits the scope', async () => {
    const w = mountIt({ data: list([]) })
    await w.findAll('[role="tab"]')[1]!.trigger('click')
    expect(w.emitted('update:scope')![0]).toEqual(['sprint'])
    await w.findAll('[role="tab"]')[0]!.trigger('click')
    expect(w.emitted('update:scope')![1]).toEqual(['all'])
  })

  it('arrow keys move between tabs and select them', async () => {
    const w = mountIt({ data: list([]) })
    await w.findAll('[role="tab"]')[0]!.trigger('keydown', { key: 'ArrowRight' })
    expect(w.emitted('update:scope')![0]).toEqual(['sprint'])
    await w.findAll('[role="tab"]')[1]!.trigger('keydown', { key: 'ArrowLeft' })
    expect(w.emitted('update:scope')![1]).toEqual(['all'])
    await w.findAll('[role="tab"]')[0]!.trigger('keydown', { key: 'ArrowLeft' }) // wraps
    expect(w.emitted('update:scope')![2]).toEqual(['sprint'])
  })

  it('only the selected tab is in the tab order', () => {
    const tabs = mountIt({ data: list([]) }).findAll('[role="tab"]')
    expect(tabs[0]!.attributes('tabindex')).toBe('0')
    expect(tabs[1]!.attributes('tabindex')).toBe('-1')
  })

  it('reads "Priya moved SCR-12 to In Review" and "Sam commented on SCR-13", each linking to the task', () => {
    const w = mountIt({ data: list([move(1), comment(2)]) })
    const rows = w.findAll('[data-activity]')
    expect(rows).toHaveLength(2)
    expect(rows[0]!.text()).toContain('Priya Nair')
    expect(rows[0]!.text()).toMatch(/moved\s+SCR-12\s+to\s+In Review/)
    expect(rows[0]!.text()).toContain('at 2026-10-17T10:00:00Z')
    expect(rows[0]!.find('a').attributes('href')).toBe('/projects/scr/issues/12')
    expect(rows[1]!.text()).toMatch(/Sam Smith\s+commented on\s+SCR-13/)
    expect(rows[1]!.find('a').attributes('href')).toBe('/projects/scr/issues/13')
  })

  it('shows no comment text, only who commented', () => {
    const w = mountIt({ data: list([{ ...comment(2), body: 'secret text' } as unknown as RecentActivityItem]) })
    expect(w.text()).not.toContain('secret text')
  })

  it('has its own empty message per tab', () => {
    expect(mountIt({ data: list([]), scope: 'all' }).text()).toMatch(/no recent activity in the last 5 days/i)
    expect(mountIt({ data: list([]), scope: 'sprint' }).text()).toMatch(/no recent activity in the last 5 days/i)
  })

  it('on the sprint tab with no active sprint says so', () => {
    const w = mountIt({ data: list([], false), scope: 'sprint' })
    expect(w.text()).toMatch(/no active sprint/i)
    expect(w.text()).not.toMatch(/no recent activity/i)
  })

  it('on the all tab without a sprint still lists activity', () => {
    const w = mountIt({ data: list([move(1)], false), scope: 'all' })
    expect(w.findAll('[data-activity]')).toHaveLength(1)
  })

  it('keeps the tabs while loading and shows a skeleton in the panel', () => {
    const w = mountIt({ loading: true })
    expect(w.findAll('[role="tab"]')).toHaveLength(2)
    expect(w.find('.skeleton').exists()).toBe(true)
  })

  it('shows the error with a Retry button, tabs still usable', async () => {
    const w = mountIt({ error: 'bad' })
    expect(w.findAll('[role="tab"]')).toHaveLength(2)
    expect(w.text()).toContain('bad')
    await w.find('[data-retry]').trigger('click')
    expect(w.emitted('retry')).toHaveLength(1)
  })

  it('reads "Priya deleted SCR-40" with the task title, and does not link to the task that is gone', () => {
    const w = mountIt({ data: list([deleted(3), move(1)]) })
    const rows = w.findAll('[data-activity]')
    expect(rows).toHaveLength(2)
    expect(rows[0]!.text()).toContain('Priya Nair')
    expect(rows[0]!.text()).toContain('deleted')
    expect(rows[0]!.text()).toContain('SCR-40')
    expect(rows[0]!.text()).toContain('Old idea')
    expect(rows[0]!.find('a').exists()).toBe(false)
    expect(rows[1]!.find('a').attributes('href')).toBe('/projects/scr/issues/12')
  })
})

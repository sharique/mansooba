// @vitest-environment happy-dom
import { shallowMount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ActivityItem from './ActivityItem.vue'
import type { ActivityEvent } from '~/types/domain.types'

vi.stubGlobal('useTimeFormatter', () => ({ formatDateTime: () => 'now' }))

const event = (e: Partial<ActivityEvent>): ActivityEvent => ({
  id: 1, issue_id: 7, actor_id: 2, actor_name: 'Alex', actor_avatar_url: null,
  issue_key: '', issue_title: '', kind: 'assignee_changed', created_at: '2026-10-14T00:00:00Z', ...e,
})

const text = (e: Partial<ActivityEvent>) => shallowMount(ActivityItem, { props: { event: event(e) } }).text()

describe('ActivityItem', () => {
  it('names the previous and new assignee', () => {
    expect(text({ old_value: 'Sam', new_value: 'Priya' })).toContain('Alex changed assignee from "Sam" to "Priya"')
  })

  it('shows "unassigned" as "nobody" on either side', () => {
    expect(text({ old_value: 'unassigned', new_value: 'Priya' })).toContain('from "nobody" to "Priya"')
    expect(text({ old_value: 'Priya', new_value: 'unassigned' })).toContain('from "Priya" to "nobody"')
  })

  it('names the sprint moved from and to, with "backlog" as "the backlog"', () => {
    expect(text({ kind: 'sprint_changed', old_value: 'backlog', new_value: 'Sprint 3' }))
      .toContain('Alex moved this from "the backlog" to "Sprint 3"')
    expect(text({ kind: 'sprint_changed', old_value: 'Sprint 3', new_value: 'backlog' }))
      .toContain('from "Sprint 3" to "the backlog"')
  })

  it('still renders old entries that lack values', () => {
    expect(text({ old_value: undefined, new_value: undefined })).toContain('Alex changed assignee')
  })

  it('leaves other kinds unchanged', () => {
    expect(text({ kind: 'status_changed', old_value: 'todo', new_value: 'done' }))
      .toContain('Alex changed status from "todo" to "done"')
    expect(text({ kind: 'comment_added' })).toContain('Alex added a comment')
  })
})

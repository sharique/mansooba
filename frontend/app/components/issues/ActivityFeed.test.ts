// @vitest-environment happy-dom
import { shallowMount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ActivityFeed from './ActivityFeed.vue'
import ActivityItem from './ActivityItem.vue'
import type { ActivityEvent, Comment } from '~/types/domain.types'

vi.stubGlobal('useTimeFormatter', () => ({ formatDateTime: (iso: string) => iso }))

const ev = (id: number, kind: string, at: string): ActivityEvent => ({
  id, issue_id: 7, actor_id: 2, actor_name: 'Alex', actor_avatar_url: null,
  issue_key: 'P-7', issue_title: 'T', kind, created_at: at,
})
const comment = (id: number, at: string): Comment => ({
  id, issue_id: 7, author_id: 2, author_name: 'Alex', body: 'hi', created_at: at, updated_at: at,
}) as Comment

const activity = [
  ev(1, 'status_changed', '2026-10-30T09:00:00Z'),
  ev(2, 'comment_added', '2026-10-30T10:00:00Z'),
  ev(3, 'issue_deleted', '2026-10-30T11:00:00Z'),
]
const shown = (w: ReturnType<typeof shallowMount>) => w.findAllComponents(ActivityItem).map(c => (c.props('event') as ActivityEvent).id)

describe('ActivityFeed', () => {
  it('on a task page leaves comment events out: the comments themselves are listed instead', () => {
    const w = shallowMount(ActivityFeed, { props: { comments: [comment(5, '2026-10-30T10:00:00Z')], activity } })
    expect(shown(w)).toEqual([1, 3])
    expect(w.text()).toContain('Alex added a comment')
  })

  it('for a person\'s own activity (no comments passed) shows the comment events too', () => {
    const w = shallowMount(ActivityFeed, { props: { comments: [], activity, showCommentEvents: true } })
    expect(shown(w)).toEqual([1, 2, 3])
  })

  it('lists oldest first by default and newest first when asked', () => {
    const oldestFirst = shallowMount(ActivityFeed, { props: { comments: [], activity, showCommentEvents: true } })
    expect(shown(oldestFirst)).toEqual([1, 2, 3])
    const newestFirst = shallowMount(ActivityFeed, { props: { comments: [], activity, showCommentEvents: true, newestFirst: true } })
    expect(shown(newestFirst)).toEqual([3, 2, 1])
  })
})

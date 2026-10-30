import { describe, expect, it } from 'vitest'
import { describeActivity } from './activityText'
import type { ActivityEvent } from '~/types/domain.types'

const ev = (e: Partial<ActivityEvent>): ActivityEvent => ({
  id: 1, issue_id: 7, actor_id: 2, actor_name: 'Alex', actor_avatar_url: null,
  issue_key: 'P-7', issue_title: 'Some task', kind: 'status_changed', created_at: '2026-10-30T00:00:00Z', ...e,
})

// One wording for every list of a person's or a task's activity, so the same event never
// reads differently in two places.
describe('describeActivity', () => {
  it('describes every kind the server records', () => {
    const d = (e: Partial<ActivityEvent>) => describeActivity(ev(e))
    expect(d({ kind: 'status_changed', old_value: 'todo', new_value: 'done' })).toBe('[P-7] Alex changed status from "todo" to "done"')
    expect(d({ kind: 'assignee_changed', old_value: 'Sam', new_value: 'unassigned' })).toBe('[P-7] Alex changed assignee from "Sam" to "nobody"')
    expect(d({ kind: 'assignee_changed' })).toBe('[P-7] Alex changed assignee')
    expect(d({ kind: 'priority_changed', old_value: 'low', new_value: 'high' })).toBe('[P-7] Alex changed priority from "low" to "high"')
    expect(d({ kind: 'sprint_changed', old_value: 'backlog', new_value: 'Sprint 3' })).toBe('[P-7] Alex moved this from "the backlog" to "Sprint 3"')
    expect(d({ kind: 'sprint_changed' })).toBe('[P-7] Alex changed sprint')
    expect(d({ kind: 'story_points_changed', old_value: '3', new_value: '5' })).toBe('[P-7] Alex changed story points from 3 to 5')
    expect(d({ kind: 'label_added', new_value: 'bug' })).toBe('[P-7] Alex added label "bug"')
    expect(d({ kind: 'label_removed', old_value: 'bug' })).toBe('[P-7] Alex removed label "bug"')
    expect(d({ kind: 'comment_added' })).toBe('[P-7] Alex added a comment')
  })

  it('says a task was deleted and names it, as the task no longer has a page', () => {
    expect(describeActivity(ev({ kind: 'issue_deleted', issue_key: 'P-17', issue_title: 'Keyboard shortcuts' })))
      .toBe('[P-17] Alex deleted the task "Keyboard shortcuts"')
  })

  it('names the file of an attachment that was added or removed', () => {
    expect(describeActivity(ev({ kind: 'attachment_added', new_value: 'plan.pdf' }))).toBe('[P-7] Alex added attachment "plan.pdf"')
    expect(describeActivity(ev({ kind: 'attachment_removed', old_value: 'plan.pdf' }))).toBe('[P-7] Alex removed attachment "plan.pdf"')
  })

  it('leaves out the key when the task is unknown, and falls back for a kind it does not know', () => {
    expect(describeActivity(ev({ kind: 'comment_added', issue_key: '' }))).toBe('Alex added a comment')
    expect(describeActivity(ev({ kind: 'something_new' }))).toBe('[P-7] Alex updated issue')
  })

  it('names the actor as "User n" when the name is missing', () => {
    expect(describeActivity(ev({ kind: 'comment_added', actor_name: '' }))).toBe('[P-7] User 2 added a comment')
  })
})

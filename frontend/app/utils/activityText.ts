import type { ActivityEvent } from '~/types/domain.types'
import { ActivityKind } from '~/types/domain.types'

// The server records a person's name, or "unassigned"; a sprint's name, or "backlog".
const who = (v?: string) => (v === 'unassigned' ? 'nobody' : v)
const where = (v?: string) => (v === 'backlog' ? 'the backlog' : v)

// The one sentence for an activity event, used by every list of a person's or a task's
// activity so the same event never reads differently in two places. An unknown kind
// (added by a newer server) gets a generic sentence.
export function describeActivity(e: ActivityEvent): string {
  const actor = e.actor_name || `User ${e.actor_id}`
  const issue = e.issue_key ? `[${e.issue_key}] ` : ''
  switch (e.kind) {
    case ActivityKind.StatusChanged: return `${issue}${actor} changed status from "${e.old_value}" to "${e.new_value}"`
    case ActivityKind.AssigneeChanged: return e.old_value && e.new_value
      ? `${issue}${actor} changed assignee from "${who(e.old_value)}" to "${who(e.new_value)}"`
      : `${issue}${actor} changed assignee`
    case ActivityKind.PriorityChanged: return `${issue}${actor} changed priority from "${e.old_value}" to "${e.new_value}"`
    case ActivityKind.SprintChanged: return e.old_value && e.new_value
      ? `${issue}${actor} moved this from "${where(e.old_value)}" to "${where(e.new_value)}"`
      : `${issue}${actor} changed sprint`
    case ActivityKind.StoryPointsChanged: return `${issue}${actor} changed story points from ${e.old_value} to ${e.new_value}`
    case ActivityKind.LabelAdded: return `${issue}${actor} added label "${e.new_value}"`
    case ActivityKind.LabelRemoved: return `${issue}${actor} removed label "${e.old_value}"`
    case ActivityKind.CommentAdded: return `${issue}${actor} added a comment`
    case ActivityKind.AttachmentAdded: return `${issue}${actor} added attachment "${e.new_value}"`
    case ActivityKind.AttachmentRemoved: return `${issue}${actor} removed attachment "${e.old_value}"`
    case ActivityKind.IssueDeleted: return `${issue}${actor} deleted the task "${e.issue_title}"`
    default: return `${issue}${actor} updated issue`
  }
}

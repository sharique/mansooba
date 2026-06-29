import type { Issue } from '~/types/domain.types'

type Status = Issue['status']
type Priority = Issue['priority']

export function statusBadgeClass(status: Status): string {
  switch (status) {
    case 'in_progress': return 'badge-primary'
    case 'in_review':   return 'badge-secondary'
    case 'done':        return 'badge-success'
    case 'todo':        return 'badge-ghost'
    case 'backlog':     return 'badge-ghost'
    default:            return 'badge-ghost'
  }
}

export function statusLabel(status: Status): string {
  switch (status) {
    case 'todo':        return 'Todo'
    case 'in_progress': return 'In Progress'
    case 'in_review':   return 'In Review'
    case 'done':        return 'Done'
    case 'backlog':     return 'Backlog'
    default:            return status
  }
}

export function priorityBadgeClass(priority: Priority): string {
  switch (priority) {
    case 'critical': return 'badge-error'
    case 'high':     return 'badge-warning'
    case 'medium':   return 'badge-info'
    case 'low':      return 'badge-ghost'
    default:         return 'badge-ghost'
  }
}

export function priorityDotClass(priority: Priority): string {
  switch (priority) {
    case 'critical': return 'bg-error'
    case 'high':     return 'bg-warning'
    case 'medium':   return 'bg-info'
    case 'low':      return 'bg-base-300'
    default:         return 'bg-base-300'
  }
}

// Left-border accent on board cards — lets users scan priority without reading text
export function priorityBorderClass(priority: Priority): string {
  switch (priority) {
    case 'critical': return 'border-l-error'
    case 'high':     return 'border-l-warning'
    case 'medium':   return 'border-l-info'
    case 'low':      return 'border-l-base-300'
    default:         return 'border-l-base-300'
  }
}

export function typeIconName(type: string): string {
  switch (type) {
    case 'epic':  return 'mdi:lightning-bolt'
    case 'story': return 'mdi:book-open-page-variant-outline'
    case 'task':  return 'mdi:check-circle-outline'
    case 'bug':   return 'mdi:bug-outline'
    default:      return 'mdi:circle-outline'
  }
}

export function typeIconClass(type: string): string {
  switch (type) {
    case 'epic':  return 'text-accent'
    case 'story': return 'text-success'
    case 'task':  return 'text-primary'
    case 'bug':   return 'text-error'
    default:      return 'text-base-content/40'
  }
}

export function statusColumnBorderClass(status: string): string {
  switch (status) {
    case 'in_progress': return 'border-t-primary'
    case 'in_review':   return 'border-t-secondary'
    case 'done':        return 'border-t-success'
    case 'todo':        return 'border-t-base-content/30'
    case 'backlog':     return 'border-t-base-content/15'
    default:            return 'border-t-base-300'
  }
}

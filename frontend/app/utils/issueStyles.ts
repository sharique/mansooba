import type { Issue } from '~/types/domain.types'

type Status = Issue['status']
type Priority = Issue['priority']

export function statusBadgeClass(status: Status): string {
  switch (status) {
    case 'in_progress': return 'badge-primary'
    case 'in_review':   return 'badge-secondary'
    case 'done':        return 'badge-success'
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
    default:         return 'badge-ghost'
  }
}

export function priorityDotClass(priority: Priority): string {
  switch (priority) {
    case 'critical': return 'bg-error'
    case 'high':     return 'bg-warning'
    case 'medium':   return 'bg-info'
    default:         return 'bg-base-300'
  }
}

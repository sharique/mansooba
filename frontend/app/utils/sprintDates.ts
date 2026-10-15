import type { Sprint } from '~/types/domain.types'

// All comparisons use YYYY-MM-DD strings, which order correctly as plain text.
const ymd = (iso?: string | null) => iso?.slice(0, 10) ?? ''

export function todayYmd(): string {
  return new Date().toISOString().slice(0, 10)
}

/** An in-progress sprint whose end date is already behind us. */
export function isOverdue(sprint: Pick<Sprint, 'status' | 'end_date'>, today = todayYmd()): boolean {
  const end = ymd(sprint.end_date)
  return sprint.status === 'active' && !!end && end < today
}

/**
 * Sprints (other than `selfId`, and not completed ones, which are history) whose
 * dates overlap start..end. Sprints that only share a boundary day are treated as
 * back-to-back, not overlapping. Sprints without both dates are ignored.
 */
export function overlappingSprints(start: string, end: string, others: Sprint[], selfId?: string): Sprint[] {
  if (!start || !end) return []
  return others.filter((o) => {
    const os = ymd(o.start_date)
    const oe = ymd(o.end_date)
    if (o.id === selfId || o.status === 'completed' || !os || !oe) return false
    return start < oe && os < end
  })
}

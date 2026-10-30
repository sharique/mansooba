import type { Issue } from '~/types/domain.types'

// Mirrors the server's board ordering (backend/internal/service/board_service.go):
// priority critical, high, medium, low, anything else last, then issue number.
// Both are tested against testdata/board-order-cases.json.
const RANK: Record<string, number> = { critical: 0, high: 1, medium: 2, low: 3 }

export function priorityRank(priority: string): number {
  return RANK[priority] ?? 4
}

// Issue keys count up with the id inside one project, so the id is the issue number.
export function sortBoardIssues<T extends Pick<Issue, 'id' | 'priority'>>(issues: T[]): T[] {
  return [...issues].sort((a, b) => priorityRank(a.priority) - priorityRank(b.priority) || a.id - b.id)
}

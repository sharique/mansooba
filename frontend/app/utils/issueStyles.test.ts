import { describe, it, expect } from 'vitest'
import {
  statusBadgeClass, statusLabel, priorityBadgeClass, priorityDotClass,
} from './issueStyles'

describe('issueStyles', () => {
  it('maps statuses to badge classes', () => {
    expect(statusBadgeClass('in_progress')).toBe('badge-primary')
    expect(statusBadgeClass('in_review')).toBe('badge-secondary')
    expect(statusBadgeClass('done')).toBe('badge-success')
    expect(statusBadgeClass('todo')).toBe('badge-ghost')
    expect(statusBadgeClass('backlog')).toBe('badge-ghost')
  })

  it('maps statuses to human labels', () => {
    expect(statusLabel('in_progress')).toBe('In Progress')
    expect(statusLabel('in_review')).toBe('In Review')
    expect(statusLabel('done')).toBe('Done')
    expect(statusLabel('todo')).toBe('Todo')
    expect(statusLabel('backlog')).toBe('Backlog')
  })

  it('maps priorities to badge classes (critical = error)', () => {
    expect(priorityBadgeClass('critical')).toBe('badge-error')
    expect(priorityBadgeClass('high')).toBe('badge-warning')
    expect(priorityBadgeClass('medium')).toBe('badge-info')
    expect(priorityBadgeClass('low')).toBe('badge-ghost')
  })

  it('maps priorities to dot background classes', () => {
    expect(priorityDotClass('critical')).toBe('bg-error')
    expect(priorityDotClass('high')).toBe('bg-warning')
    expect(priorityDotClass('medium')).toBe('bg-info')
    expect(priorityDotClass('low')).toBe('bg-base-300')
  })
})

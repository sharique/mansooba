import { describe, it, expect } from 'vitest'
import {
  statusBadgeClass, statusLabel, priorityBadgeClass, priorityDotClass,
  priorityBorderClass, typeIconName, typeIconClass, statusColumnBorderClass,
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

  it('maps priorities to left-border classes', () => {
    expect(priorityBorderClass('critical')).toBe('border-l-error')
    expect(priorityBorderClass('high')).toBe('border-l-warning')
    expect(priorityBorderClass('medium')).toBe('border-l-info')
    expect(priorityBorderClass('low')).toBe('border-l-base-300')
  })

  it('maps issue types to MDI icon names', () => {
    expect(typeIconName('epic')).toBe('mdi:lightning-bolt')
    expect(typeIconName('story')).toBe('mdi:book-open-page-variant-outline')
    expect(typeIconName('task')).toBe('mdi:check-circle-outline')
    expect(typeIconName('bug')).toBe('mdi:bug-outline')
    expect(typeIconName('unknown')).toBe('mdi:circle-outline')
  })

  it('maps issue types to icon color classes', () => {
    expect(typeIconClass('epic')).toBe('text-accent')
    expect(typeIconClass('story')).toBe('text-success')
    expect(typeIconClass('task')).toBe('text-primary')
    expect(typeIconClass('bug')).toBe('text-error')
  })

  it('maps statuses to column top-border classes', () => {
    expect(statusColumnBorderClass('in_progress')).toBe('border-t-primary')
    expect(statusColumnBorderClass('in_review')).toBe('border-t-secondary')
    expect(statusColumnBorderClass('done')).toBe('border-t-success')
  })
})

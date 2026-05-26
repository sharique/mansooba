import { describe, it, expect } from 'vitest'
import { projectNavLinks } from './Sidebar.vue'

describe('projectNavLinks', () => {
  it('returns the six project sections for a key', () => {
    const links = projectNavLinks('apollo')
    expect(links.map(l => l.label)).toEqual(
      ['Overview', 'Board', 'Backlog', 'Sprints', 'Reports', 'Settings'],
    )
    expect(links[1]).toEqual({ label: 'Board', to: '/projects/apollo/board', icon: 'mdi:view-column-outline' })
  })

  it('generates correct routes for all six sections', () => {
    const links = projectNavLinks('demo')
    expect(links[0].to).toBe('/projects/demo')
    expect(links[1].to).toBe('/projects/demo/board')
    expect(links[2].to).toBe('/projects/demo/backlog')
    // Sprints intentionally shares the backlog route until a dedicated page exists
    expect(links[3].to).toBe('/projects/demo/backlog')
    expect(links[4].to).toBe('/projects/demo/reports')
    expect(links[5].to).toBe('/projects/demo/settings')
  })

  it('uses the provided key in every route', () => {
    const links = projectNavLinks('xyz')
    expect(links.every(l => l.to.includes('/projects/xyz'))).toBe(true)
  })
})

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
})

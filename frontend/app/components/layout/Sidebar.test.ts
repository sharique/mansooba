import { describe, it, expect } from 'vitest'
import { projectNavLinks, type NavLink } from './Sidebar.vue'

describe('projectNavLinks', () => {
  it('returns the five project sections for a key', () => {
    const links = projectNavLinks('apollo')
    expect(links.map(l => l.label)).toEqual(
      ['Overview', 'Board', 'Backlog & Sprints', 'Reports', 'Settings'],
    )
    expect(links[1]).toEqual({ label: 'Board', to: '/projects/apollo/board', icon: 'mdi:view-column-outline' })
  })

  it('generates correct routes for all five sections', () => {
    const links = projectNavLinks('demo')
    expect(links[0].to).toBe('/projects/demo')
    expect(links[1].to).toBe('/projects/demo/board')
    expect(links[2].to).toBe('/projects/demo/backlog')
    expect(links[3].to).toBe('/projects/demo/reports')
    expect(links[4].to).toBe('/projects/demo/settings')
  })

  it('has no two entries pointing at the same page', () => {
    const links = projectNavLinks('demo')
    expect(new Set(links.map(l => l.to)).size).toBe(links.length)
  })

  it('uses the provided key in every route', () => {
    const links = projectNavLinks('xyz')
    expect(links.every(l => l.to.includes('/projects/xyz'))).toBe(true)
  })
})

describe('system nav links (admin only)', () => {
  const systemLinks: NavLink[] = [
    { label: 'System Settings', to: '/system/settings', icon: 'mdi:cog-outline' },
    { label: 'User Management', to: '/system/users',    icon: 'mdi:account-group-outline' },
    { label: 'Create User',     to: '/system/createuser', icon: 'mdi:account-plus-outline' },
    { label: 'System Logs',     to: '/system/logs',      icon: 'mdi:text-box-search-outline' },
  ]

  it('contains four links', () => {
    expect(systemLinks).toHaveLength(4)
  })

  it('system settings link points to /system/settings', () => {
    expect(systemLinks[0].to).toBe('/system/settings')
    expect(systemLinks[0].label).toBe('System Settings')
  })

  it('user management link points to /system/users', () => {
    expect(systemLinks[1].to).toBe('/system/users')
    expect(systemLinks[1].label).toBe('User Management')
  })

  it('create user link points to /system/createuser', () => {
    expect(systemLinks[2].to).toBe('/system/createuser')
    expect(systemLinks[2].label).toBe('Create User')
  })

  it('system logs link points to /system/logs', () => {
    expect(systemLinks[3].to).toBe('/system/logs')
    expect(systemLinks[3].label).toBe('System Logs')
  })
})

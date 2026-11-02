// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, it, expect, vi } from 'vitest'
import { projectNavLinks, systemNavLinks } from './Sidebar.vue'

describe('projectNavLinks', () => {
  it('returns the seven project sections for a key, Members after Sprints', () => {
    const links = projectNavLinks('apollo')
    expect(links.map(l => l.label)).toEqual(
      ['Overview', 'Sprint board', 'Backlog', 'Sprints', 'Members', 'Reports', 'Settings'],
    )
    expect(links[4]).toEqual({ label: 'Members', to: '/projects/apollo/members', icon: 'mdi:account-group-outline' })
    expect(links[1]).toEqual({ label: 'Sprint board', to: '/projects/apollo/board', icon: 'mdi:view-column-outline' })
  })

  it('generates correct routes for all sections', () => {
    const links = projectNavLinks('demo')
    expect(links.map(l => l.to)).toEqual([
      '/projects/demo',
      '/projects/demo/board',
      '/projects/demo/backlog',
      '/projects/demo/sprints',
      '/projects/demo/members',
      '/projects/demo/reports',
      '/projects/demo/settings',
    ])
  })

  it('gives every entry an icon', () => {
    expect(projectNavLinks('demo').every(l => l.icon.startsWith('mdi:'))).toBe(true)
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
  it('contains the five links, the abandoned-projects page last', () => {
    expect(systemNavLinks().map(l => l.to)).toEqual([
      '/system/settings', '/system/users', '/system/createuser', '/system/logs', '/system/projects',
    ])
  })

  it('labels the abandoned-projects entry "Projects without an admin"', () => {
    const link = systemNavLinks().find(l => l.to === '/system/projects')
    expect(link?.label).toBe('Projects without an admin')
    expect(link?.icon.startsWith('mdi:')).toBe(true)
  })

  it('keeps the existing links and labels', () => {
    expect(systemNavLinks().slice(0, 4).map(l => l.label)).toEqual(
      ['System Settings', 'User Management', 'Create User', 'System Logs'],
    )
  })
})

describe('Sidebar system section', () => {
  const mountIt = async (isAdmin: boolean) => {
    vi.resetModules()
    vi.stubGlobal('useRoute', () => ({ path: '/', params: {} }))
    vi.stubGlobal('onMounted', () => {})
    vi.doMock('~/stores/projects.store', () => ({ useProjectsStore: () => ({ projects: [], fetchAll: vi.fn() }) }))
    vi.doMock('~/stores/auth.store', () => ({ useAuthStore: () => ({ isAdmin }) }))
    const { default: Sidebar } = await import('./Sidebar.vue')
    return mount(Sidebar, { global: { stubs: { NuxtLink: { props: ['to'], template: '<a :href="to"><slot /></a>' }, Icon: true } } })
  }

  it('shows "Projects without an admin" to a system administrator', async () => {
    const w = await mountIt(true)
    expect(w.find('a[href="/system/projects"]').text()).toContain('Projects without an admin')
  })

  it('hides it from everyone else', async () => {
    const w = await mountIt(false)
    expect(w.find('a[href="/system/projects"]').exists()).toBe(false)
  })
})

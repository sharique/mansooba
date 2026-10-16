// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import MemberList from './MemberList.vue'

const members = [
  { user_id: 1, name: 'Sam', email: 'sam@example.com', username: 'sam', role: 'admin', is_active: true },
  { user_id: 2, name: 'Legacy', email: 'old@example.com', username: '', role: 'member', is_active: true },
]

vi.mock('~/services/projects.service', () => ({
  projectsService: {
    listMembers: vi.fn(async () => members),
    addMember: vi.fn(),
    removeMember: vi.fn(),
  },
}))
vi.mock('~/stores/auth.store', () => ({ useAuthStore: () => ({ user: { id: 1 } }) }))
vi.stubGlobal('useToast', () => ({ showSuccess: vi.fn(), showError: vi.fn() }))

describe('MemberList', () => {
  it('shows @username next to the name, and nothing for an account that has none yet', async () => {
    const w = mount(MemberList, { props: { projectKey: 'abc', ownerId: 1 } })
    await flushPromises()
    const rows = w.findAll('tbody tr')
    expect(rows).toHaveLength(2)
    expect(rows[0].text()).toContain('@sam')
    expect(rows[0].text()).toContain('sam@example.com')
    expect(rows[1].text()).not.toContain('@old')
    expect(w.find('thead').text()).toContain('Username')
  })
})

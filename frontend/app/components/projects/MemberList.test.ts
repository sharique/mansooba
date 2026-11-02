// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import MemberList from './MemberList.vue'
import { useProjectsStore } from '~/stores/projects.store'
import { stubPermissions, ADMIN, MEMBER, VIEWER } from '../../../test-helpers/permissions'

const perms = stubPermissions(ADMIN, 'admin')
const row = (user_id: number, name: string, role: string, is_active = true) =>
  ({ user_id, name, email: `${name.toLowerCase()}@example.com`, username: name.toLowerCase(), role, is_active })

const state = vi.hoisted(() => ({ members: [] as unknown[] }))
const changeRole = vi.hoisted(() => vi.fn())

vi.mock('~/services/projects.service', () => ({
  projectsService: {
    listMembers: vi.fn(async () => state.members),
    addMember: vi.fn(),
    removeMember: vi.fn(),
    changeMemberRole: changeRole,
  },
}))
vi.mock('~/stores/auth.store', () => ({ useAuthStore: () => ({ user: { id: 1 } }) }))
const toast = { showSuccess: vi.fn(), showError: vi.fn() }
vi.stubGlobal('useToast', () => toast)
vi.stubGlobal('useProjectsStore', useProjectsStore)

const mountIt = async () => {
  const w = mount(MemberList, { props: { projectKey: 'abc' } })
  await flushPromises()
  return w
}
const roleSelect = (w: Awaited<ReturnType<typeof mountIt>>, name: string) => w.find(`select[aria-label="Role of ${name}"]`)

describe('MemberList', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    perms.permissions = ADMIN
    perms.myRole = 'admin'
    state.members = [row(1, 'Sam', 'admin'), { ...row(2, 'Legacy', 'member'), username: '', email: 'old@example.com' }]
    changeRole.mockReset()
    toast.showError.mockReset()
    toast.showSuccess.mockReset()
  })

  it('shows @username next to the name, and nothing for an account that has none yet', async () => {
    const w = await mountIt()
    const rows = w.findAll('tbody tr')
    expect(rows).toHaveLength(2)
    expect(rows[0].text()).toContain('@sam')
    expect(rows[0].text()).toContain('sam@example.com')
    expect(rows[1].text()).not.toContain('@old')
    expect(w.find('thead').text()).toContain('Username')
  })

  describe('as someone who can manage people', () => {
    it('shows a role dropdown per person, with the current role selected', async () => {
      const w = await mountIt()
      expect((roleSelect(w, 'Legacy').element as HTMLSelectElement).value).toBe('member')
      expect(roleSelect(w, 'Legacy').attributes('disabled')).toBeUndefined()
    })

    it('disables the dropdown of the only active admin and says why', async () => {
      const w = await mountIt()
      const sel = roleSelect(w, 'Sam')
      expect(sel.attributes('disabled')).toBeDefined()
      expect(sel.attributes('title')).toBe('A project needs at least one active admin')
    })

    it('does not treat an inactive admin as the one that counts', async () => {
      state.members = [row(1, 'Sam', 'admin'), row(3, 'Gone', 'admin', false)]
      const w = await mountIt()
      expect(roleSelect(w, 'Sam').attributes('disabled')).toBeDefined()
      expect(roleSelect(w, 'Gone').attributes('disabled')).toBeUndefined()
    })

    it('enables every admin dropdown when there are two active admins', async () => {
      state.members = [row(1, 'Sam', 'admin'), row(3, 'Pat', 'admin')]
      const w = await mountIt()
      expect(roleSelect(w, 'Sam').attributes('disabled')).toBeUndefined()
      expect(roleSelect(w, 'Pat').attributes('disabled')).toBeUndefined()
    })

    it('has no Remove button on their own row but has one on the others', async () => {
      const w = await mountIt()
      const rows = w.findAll('tbody tr')
      expect(rows[0].text()).not.toContain('Remove')
      expect(rows[1].text()).toContain('Remove')
    })

    it('changing a role calls the server and shows the new role', async () => {
      changeRole.mockResolvedValueOnce(row(2, 'Legacy', 'viewer'))
      const w = await mountIt()
      await roleSelect(w, 'Legacy').setValue('viewer')
      await flushPromises()
      expect(changeRole).toHaveBeenCalledWith('abc', 2, 'viewer')
      expect(toast.showSuccess).toHaveBeenCalled()
      expect((roleSelect(w, 'Legacy').element as HTMLSelectElement).value).toBe('viewer')
    })

    it('shows the server reason and puts the old role back when the change is refused', async () => {
      changeRole.mockRejectedValueOnce({ status: 409, data: { message: 'A project needs at least one active admin' } })
      const w = await mountIt()
      await roleSelect(w, 'Legacy').setValue('viewer')
      await flushPromises()
      expect(toast.showError).toHaveBeenCalledWith('A project needs at least one active admin')
      expect((roleSelect(w, 'Legacy').element as HTMLSelectElement).value).toBe('member')
    })

    it('refreshes the permissions when the person changes their own role', async () => {
      state.members = [row(1, 'Sam', 'admin'), row(3, 'Pat', 'admin')]
      changeRole.mockResolvedValueOnce(row(1, 'Sam', 'member'))
      const refresh = vi.spyOn(useProjectsStore(), 'refreshPermissions').mockResolvedValue()
      const w = await mountIt()
      await roleSelect(w, 'Sam').setValue('member')
      await flushPromises()
      expect(refresh).toHaveBeenCalledWith('abc')
    })

    it('offers the invite form', async () => {
      const w = await mountIt()
      expect(w.text()).toContain('Invite member')
    })
  })

  describe.each([['member', MEMBER], ['viewer', VIEWER]])('as a %s', (role, perm) => {
    beforeEach(() => {
      perms.permissions = perm
      perms.myRole = role
    })

    it('sees role badges, no dropdown, no Remove, no invite form', async () => {
      const w = await mountIt()
      expect(w.find('select[aria-label^="Role of"]').exists()).toBe(false)
      expect(w.find('tbody').text()).toContain('admin')
      expect(w.text()).not.toContain('Remove')
      expect(w.text()).not.toContain('Invite member')
    })
  })
})

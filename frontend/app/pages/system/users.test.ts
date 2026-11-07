// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils'
import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import UsersPage from './users.vue'

const navigate = vi.fn()
const auth = { isAdmin: true }
const patchUser = vi.fn()
const fetchUsers = vi.fn().mockResolvedValue(undefined)

const people = ref([
  { id: 1, name: 'Sys Admin', email: 'sys@example.com', is_admin: true, is_active: true, created_at: '' },
  { id: 2, name: 'Plain Person', email: 'plain@example.com', is_admin: false, is_active: true, created_at: '' },
])

vi.stubGlobal('navigateTo', navigate)
vi.mock('~/stores/auth.store', () => ({ useAuthStore: () => auth }))
vi.stubGlobal('onMounted', (fn: () => void) => fn())
vi.mock('~/composables/useAdminUsers', () => ({
  useAdminUsers: () => ({
    users: people, total: ref(2), page: ref(1), loading: ref(false), error: ref(''),
    fetchUsers, patchUser,
  }),
}))

const mountIt = () => mount(UsersPage)
const row = (w: ReturnType<typeof mount>, email: string) => w.findAll('tbody tr').find(r => r.text().includes(email))!

describe('system/users page: system administrator wording', () => {
  beforeEach(() => {
    auth.isAdmin = true
    patchUser.mockReset().mockResolvedValue({})
    fetchUsers.mockClear()
  })

  it('labels the role as system administrator or user, not as project roles', async () => {
    const w = mountIt()
    await flushPromises()
    expect(row(w, 'sys@example.com').find('.badge').text()).toBe('System administrator')
    expect(row(w, 'plain@example.com').find('.badge').text()).toBe('User')
    expect(w.text()).not.toMatch(/\bMember\b/)
  })

  it('offers to make someone a system administrator, and to remove it, in words', async () => {
    const w = mountIt()
    await flushPromises()
    const buttons = (email: string) => row(w, email).findAll('button').map(b => b.text())
    expect(buttons('plain@example.com')).toContain('Make system administrator')
    expect(buttons('sys@example.com')).toContain('Remove system administrator')
  })

  it('makes someone a system administrator straight away', async () => {
    const w = mountIt()
    await flushPromises()
    await row(w, 'plain@example.com').findAll('button').find(b => b.text() === 'Make system administrator')!.trigger('click')
    await flushPromises()
    expect(patchUser).toHaveBeenCalledWith(2, { is_admin: true })
  })

  it('asks before removing system administrator access', async () => {
    const w = mountIt()
    await flushPromises()
    await row(w, 'sys@example.com').findAll('button').find(b => b.text() === 'Remove system administrator')!.trigger('click')
    expect(patchUser).not.toHaveBeenCalled()
    expect(w.find('dialog h3').text()).toBe('Remove system administrator access?')
  })

  it('sends someone who is not a system administrator away', async () => {
    auth.isAdmin = false
    mountIt()
    await flushPromises()
    expect(navigate).toHaveBeenCalledWith('/')
    expect(fetchUsers).not.toHaveBeenCalled()
  })
})

describe('system/users page: refused changes', () => {
  const refuse = { status: 409, message: '[PATCH] "/api/v1/admin/users/1": 409 Conflict', data: { message: 'Cannot remove the last active admin. Promote another user first.' } }

  beforeEach(() => {
    auth.isAdmin = true
    patchUser.mockReset().mockRejectedValue(refuse)
  })

  it('shows the server reason in the dialog and keeps it open when the last administrator would go', async () => {
    const w = mountIt()
    await flushPromises()
    await row(w, 'sys@example.com').findAll('button').find(b => b.text() === 'Remove system administrator')!.trigger('click')
    await w.find('dialog .btn-error').trigger('click')
    await flushPromises()
    const dialog = w.find('dialog')
    expect(dialog.text()).toContain('Cannot remove the last active admin. Promote another user first.')
    expect(dialog.text()).not.toContain('409 Conflict')
    expect(people.value[0]!.is_admin).toBe(true)
  })

  it('shows the server reason, not the raw request text, when a change without a dialog is refused', async () => {
    patchUser.mockRejectedValue({ ...refuse, data: { message: 'Some other reason' } })
    const w = mountIt()
    await flushPromises()
    await row(w, 'plain@example.com').findAll('button').find(b => b.text() === 'Make system administrator')!.trigger('click')
    await flushPromises()
    expect(w.text()).toContain('Some other reason')
    expect(w.text()).not.toContain('409 Conflict')
  })
})

// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AdminStep from './AdminStep.vue'

const completeAdmin = vi.fn()
const checkUsername = vi.fn()

vi.mock('~/stores/setup.store', () => ({ useSetupStore: () => ({ completeAdmin: (...a: unknown[]) => completeAdmin(...a) }) }))
vi.mock('~/services/admin.service', () => ({ adminService: { checkUsername: (u: string) => checkUsername(u) } }))

async function fill(w: ReturnType<typeof mount>) {
  await w.find('#admin-name').setValue('Root')
  await w.find('#admin-email').setValue('admin@example.com')
  await w.find('#admin-password').setValue('Passw0rd!x')
  await w.find('#admin-password').trigger('input')
}

describe('first administrator step: username', () => {
  beforeEach(() => {
    completeAdmin.mockReset().mockResolvedValue(undefined)
    checkUsername.mockReset()
  })

  it('pre-fills from the email; admin@ becomes admin2 because admin is reserved', async () => {
    const w = mount(AdminStep)
    await fill(w)
    expect((w.find('#admin-username').element as HTMLInputElement).value).toBe('admin2')
  })

  it('checks only the rules: no availability request (no account to clash with, no token yet)', async () => {
    vi.useFakeTimers()
    const w = mount(AdminStep)
    await fill(w)
    await w.find('#admin-username').setValue('root_user')
    await vi.advanceTimersByTimeAsync(1000)
    expect(checkUsername).not.toHaveBeenCalled()
    vi.useRealTimers()
  })

  it('sends the username with the account', async () => {
    const w = mount(AdminStep)
    await fill(w)
    await w.find('#admin-username').setValue('Root_User')
    await w.find('button.btn-primary').trigger('click')
    await flushPromises()
    expect(completeAdmin).toHaveBeenCalledWith({ full_name: 'Root', email: 'admin@example.com', password: 'Passw0rd!x', username: 'Root_User' })
  })

  it('does not allow a reserved or invalid username', async () => {
    const w = mount(AdminStep)
    await fill(w)
    await w.find('#admin-username').setValue('admin')
    expect(w.find('button.btn-primary').attributes('disabled')).toBeDefined()
    expect(w.text()).toContain('That username is reserved.')
  })

  it('shows a server rejection on the username field', async () => {
    completeAdmin.mockRejectedValue({ statusCode: 400, data: { message: 'That username is reserved.' } })
    const w = mount(AdminStep)
    await fill(w)
    await w.find('#admin-username').setValue('root_user')
    await w.find('button.btn-primary').trigger('click')
    await flushPromises()
    expect(w.find('[data-testid="username-status"]').text()).toBe('That username is reserved.')
  })
})

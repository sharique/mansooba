// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import UserStep from './UserStep.vue'

const completeUser = vi.fn()
const checkUsername = vi.fn()

vi.mock('~/stores/setup.store', () => ({ useSetupStore: () => ({ completeUser: (...a: unknown[]) => completeUser(...a), skipUser: vi.fn() }) }))
vi.mock('~/services/admin.service', () => ({ adminService: { checkUsername: (u: string) => checkUsername(u) } }))

async function fill(w: ReturnType<typeof mount>) {
  await w.find('#user-name').setValue('Demo User')
  await w.find('#user-email').setValue('demo@example.com')
  await w.find('#user-password').setValue('Passw0rd!x')
  await w.find('#user-password').trigger('input')
}

describe('wizard team-member step: username', () => {
  beforeEach(() => {
    completeUser.mockReset().mockResolvedValue(undefined)
    checkUsername.mockReset().mockResolvedValue({ available: true, reason: '', message: '' })
  })

  it('pre-fills from the email and asks the server whether it is free once typing pauses', async () => {
    vi.useFakeTimers()
    const w = mount(UserStep)
    await fill(w)
    expect((w.find('#user-username').element as HTMLInputElement).value).toBe('demo')
    await vi.advanceTimersByTimeAsync(500)
    expect(checkUsername).toHaveBeenCalledWith('demo')
    vi.useRealTimers()
  })

  it('sends the username with the account', async () => {
    const w = mount(UserStep)
    await fill(w)
    await w.find('#user-username').setValue('demo_user')
    await w.find('button.btn-primary').trigger('click')
    await flushPromises()
    expect(completeUser).toHaveBeenCalledWith({ full_name: 'Demo User', email: 'demo@example.com', password: 'Passw0rd!x', username: 'demo_user' })
  })

  it('shows "taken" from the server on the username field rather than "email already registered"', async () => {
    completeUser.mockRejectedValue({ statusCode: 409, data: { message: 'That username is already taken.' } })
    const w = mount(UserStep)
    await fill(w)
    await w.find('button.btn-primary').trigger('click')
    await flushPromises()
    expect(w.find('[data-testid="username-status"]').text()).toBe('That username is already taken.')
    expect(w.text()).not.toContain('This email is already registered.')
  })

  it('still reports a duplicate email as before', async () => {
    completeUser.mockRejectedValue({ statusCode: 409, data: { message: 'Email already registered.' } })
    const w = mount(UserStep)
    await fill(w)
    await w.find('button.btn-primary').trigger('click')
    await flushPromises()
    expect(w.text()).toContain('This email is already registered.')
  })
})

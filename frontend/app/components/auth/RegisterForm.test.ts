// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import RegisterForm from './RegisterForm.vue'

const register = vi.fn()
const checkUsername = vi.fn()

vi.mock('~/services/auth.service', () => ({ authService: { register: (...a: unknown[]) => register(...a) } }))
vi.mock('~/services/admin.service', () => ({ adminService: { checkUsername: (u: string) => checkUsername(u) } }))

async function fill(w: ReturnType<typeof mount>) {
  await w.find('input[placeholder="Alice Smith"]').setValue('Sam Smith')
  await w.find('input[type=email]').setValue('Sam-Smith@example.com')
  await w.find('input[type=password]').setValue('Passw0rd!x')
}

describe('RegisterForm: username', () => {
  beforeEach(() => {
    register.mockReset().mockResolvedValue({})
    checkUsername.mockReset().mockResolvedValue({ available: true, reason: '', message: '' })
  })

  it('shows a username field pre-filled from the email', async () => {
    const w = mount(RegisterForm)
    await fill(w)
    expect((w.find('#register-username').element as HTMLInputElement).value).toBe('samsmith')
  })

  it('sends the pre-filled username, or the edited one', async () => {
    const w = mount(RegisterForm)
    await fill(w)
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(register).toHaveBeenLastCalledWith('Sam-Smith@example.com', 'Passw0rd!x', 'Sam Smith', 'samsmith')

    await w.find('#register-username').setValue('Sammy_S')
    await w.find('input[type=email]').setValue('other@example.com') // an edited name stays
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(register).toHaveBeenLastCalledWith('other@example.com', 'Passw0rd!x', 'Sam Smith', 'Sammy_S')
  })

  it('does not submit a username that breaks the rules', async () => {
    const w = mount(RegisterForm)
    await fill(w)
    await w.find('#register-username').setValue('admin')
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(register).not.toHaveBeenCalled()
    expect(w.text()).toContain('That username is reserved.')
  })

  it('shows a server rejection (taken) on the username field, not as a general error', async () => {
    register.mockRejectedValue({ data: { message: 'That username is already taken.' } })
    const w = mount(RegisterForm)
    await fill(w)
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(w.find('[data-testid="username-status"]').text()).toBe('That username is already taken.')
    expect(w.find('.alert-error').exists()).toBe(false)
  })

  it('still reports other failures as a general error', async () => {
    register.mockRejectedValue({ data: { message: 'email already registered' } })
    const w = mount(RegisterForm)
    await fill(w)
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(w.find('.alert-error').text()).toContain('email already registered')
  })
})

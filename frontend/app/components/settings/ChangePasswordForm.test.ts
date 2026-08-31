// @vitest-environment happy-dom
// T016 (012-change-password): confirm-password mismatch guard.
//
// This test targets a NOT-YET-CREATED component,
// components/settings/ChangePasswordForm.vue — the expected TDD red state.
// See specs/012-change-password/IMPLEMENTATION_GUIDE.md for the exact
// data-testid contract this test relies on (current-password, new-password,
// confirm-password, submit, mismatch-error).
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, it, expect, vi } from 'vitest'
import ChangePasswordForm from './ChangePasswordForm.vue'

const mockChangePassword = vi.fn()
const mockShowSuccess = vi.fn()
const mockShowError = vi.fn()

vi.stubGlobal('useAuthStore', () => ({ changePassword: mockChangePassword }))
vi.stubGlobal('useToast', () => ({ showSuccess: mockShowSuccess, showError: mockShowError }))

describe('ChangePasswordForm', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockChangePassword.mockReset()
    mockShowSuccess.mockReset()
    mockShowError.mockReset()
  })

  it('blocks submission and shows an inline error when new password and confirmation do not match', async () => {
    const w = mount(ChangePasswordForm)

    await w.find('[data-testid="current-password"]').setValue('OldPassword1')
    await w.find('[data-testid="new-password"]').setValue('NewPassword2')
    await w.find('[data-testid="confirm-password"]').setValue('NewPasswordDifferent3')
    await w.find('form').trigger('submit')

    expect(w.find('[data-testid="mismatch-error"]').exists()).toBe(true)
    expect(mockChangePassword).not.toHaveBeenCalled()
  })

  it('submits when new password and confirmation match', async () => {
    mockChangePassword.mockResolvedValue(undefined)
    const w = mount(ChangePasswordForm)

    await w.find('[data-testid="current-password"]').setValue('OldPassword1')
    await w.find('[data-testid="new-password"]').setValue('NewPassword2')
    await w.find('[data-testid="confirm-password"]').setValue('NewPassword2')
    await w.find('form').trigger('submit')

    expect(mockChangePassword).toHaveBeenCalledWith('OldPassword1', 'NewPassword2')
  })

  it('does not call the backend when confirmation is empty', async () => {
    const w = mount(ChangePasswordForm)

    await w.find('[data-testid="current-password"]').setValue('OldPassword1')
    await w.find('[data-testid="new-password"]').setValue('NewPassword2')
    await w.find('form').trigger('submit')

    expect(mockChangePassword).not.toHaveBeenCalled()
  })
})

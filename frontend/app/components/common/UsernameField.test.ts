// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { useUsernameField } from '~/composables/useUsernameField'
import UsernameField from './UsernameField.vue'

function mountField(check: Parameters<typeof useUsernameField>[1]['check'], email = '') {
  const emailRef = ref(email)
  const field = useUsernameField(emailRef, { check })
  const w = mount(UsernameField, { props: { field, id: 'u' } })
  return { w, field, emailRef }
}

describe('UsernameField', () => {
  it('explains what the field is for and that it is permanent', () => {
    const { w } = mountField(null)
    expect(w.text()).toContain('Used for @mentions. Cannot be changed later.')
    expect(w.find('input').attributes('autocomplete')).toBe('off')
  })

  it('shows the suggestion derived from the email', async () => {
    const { w, emailRef } = mountField(null)
    emailRef.value = 'sam-smith@example.com'
    await w.vm.$nextTick()
    expect((w.find('input').element as HTMLInputElement).value).toBe('samsmith')
  })

  it('shows rule errors as the person types', async () => {
    const { w } = mountField(null)
    await w.find('input').setValue('ab')
    expect(w.find('[data-testid="username-status"]').text()).toMatch(/3 to 30/)
    await w.find('input').setValue('admin')
    expect(w.find('[data-testid="username-status"]').text()).toMatch(/reserved/)
    expect(w.find('input').classes()).toContain('input-error')
  })

  it('shows available / taken from the server verdict and nothing when the check fails', async () => {
    vi.useFakeTimers()
    const check = vi.fn()
      .mockResolvedValueOnce({ available: true, reason: '', message: '' })
      .mockResolvedValueOnce({ available: false, reason: 'taken', message: 'That username is already taken.' })
      .mockRejectedValueOnce(new Error('offline'))
    const { w } = mountField(check)
    const input = w.find('input')

    await input.setValue('fresh_name')
    await vi.advanceTimersByTimeAsync(500)
    await w.vm.$nextTick()
    expect(w.find('[data-testid="username-status"]').text()).toBe('Available')

    await input.setValue('sam')
    await vi.advanceTimersByTimeAsync(500)
    await w.vm.$nextTick()
    expect(w.find('[data-testid="username-status"]').text()).toBe('That username is already taken.')

    await input.setValue('another_one')
    await vi.advanceTimersByTimeAsync(500)
    await w.vm.$nextTick()
    expect(w.find('[data-testid="username-status"]').exists()).toBe(false)
    vi.useRealTimers()
  })
})

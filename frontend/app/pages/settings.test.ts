// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import Settings from './settings.vue'

vi.mock('~/stores/auth.store', () => ({
  useAuthStore: () => ({
    user: { id: 1, name: 'Sam', email: 'sam@example.com' },
    profile: { id: 1, name: 'Sam', email: 'sam@example.com', username: 'sam_the_user', timezone: '', avatar_url: '' },
    updateProfile: vi.fn(),
    fetchMe: vi.fn(),
  }),
}))
vi.mock('~/services/auth.service', () => ({ authService: {} }))
vi.stubGlobal('useToast', () => ({ showSuccess: vi.fn(), showError: vi.fn() }))

describe('settings page: username', () => {
  it('shows the username read-only with a note that it cannot be changed', () => {
    const w = mount(Settings, {
      global: { stubs: { UserAvatar: true, ChangePasswordForm: true, ActivityItem: true } },
    })
    const input = w.find('[data-testid="profile-username"]')
    expect(input.exists()).toBe(true)
    expect((input.element as HTMLInputElement).value).toBe('sam_the_user')
    expect(input.attributes('disabled')).toBeDefined()
    expect(w.text()).toContain('Usernames cannot be changed.')
  })
})

// @vitest-environment happy-dom
// Every unauthenticated page must show the source/licence notice.
import { shallowMount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AppSourceNotice from '../components/common/AppSourceNotice.vue'

vi.stubGlobal('definePageMeta', () => {})
import ResetPasswordPage from './reset-password.vue'

vi.stubGlobal('useRoute', () => ({ query: { token: 'test-token' } }))
vi.mock('~/services/auth.service', () => ({
  authService: { resetPassword: vi.fn() },
}))
vi.mock('~/stores/about.store', () => ({
  useAboutStore: () => ({
    version: '1.4.0',
    sourceUrl: 'https://github.com/sharique/mansooba',
    license: 'AGPL-3.0-only',
    licenseUrl: 'https://www.gnu.org/licenses/agpl-3.0.html',
    fetch: vi.fn(),
  }),
}))

describe('reset-password page', () => {
  it('renders the source and licence notice', () => {
    const w = shallowMount(ResetPasswordPage, {
      global: { components: { AppSourceNotice }, stubs: { AppSourceNotice: false, NuxtLink: true } },
    })
    expect(w.find('[data-testid="source-notice"]').exists()).toBe(true)
  })
})

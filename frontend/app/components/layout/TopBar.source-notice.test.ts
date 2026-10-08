// @vitest-environment happy-dom
// TopBar renders AppSourceNotice. TopBar is chosen over
// Sidebar because the sidebar is a collapsed drawer below the `lg`
// breakpoint, so a notice there would need two taps on
// a phone; TopBar renders unconditionally at every screen width.
import { shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import TopBar from './TopBar.vue'
import AppSourceNotice from '../common/AppSourceNotice.vue'

vi.stubGlobal('useRouter', () => ({ push: vi.fn() }))
vi.mock('~/stores/auth.store', () => ({
  useAuthStore: () => ({ profile: null, user: null }),
}))
vi.mock('~/services/auth.service', () => ({
  authService: { logout: vi.fn() },
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

describe('TopBar', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders the source and licence notice', () => {
    const w = shallowMount(TopBar, {
      global: { components: { AppSourceNotice }, stubs: { AppSourceNotice: false, Icon: true, UserAvatar: true } },
    })
    expect(w.find('[data-testid="source-notice"]').exists()).toBe(true)
  })
})

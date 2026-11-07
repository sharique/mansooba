// @vitest-environment happy-dom
// The source and licence notice moved from the top bar to the page footer
// (see Footer.test.ts); the top bar must no longer render it.
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

  it('no longer renders the source and licence notice (it is in the footer)', () => {
    const w = shallowMount(TopBar, {
      global: { components: { AppSourceNotice }, stubs: { AppSourceNotice: false, Icon: true, UserAvatar: true } },
    })
    expect(w.find('[data-testid="source-notice"]').exists()).toBe(false)
  })
})

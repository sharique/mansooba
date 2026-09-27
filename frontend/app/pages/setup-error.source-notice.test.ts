// @vitest-environment happy-dom
// Every unauthenticated page must show the source/licence notice.
import { shallowMount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AppSourceNotice from '../components/common/AppSourceNotice.vue'

vi.stubGlobal('definePageMeta', () => {})
import SetupErrorPage from './setup-error.vue'

vi.mock('~/stores/about.store', () => ({
  useAboutStore: () => ({
    version: '1.4.0',
    sourceUrl: 'https://github.com/sharique/mansooba',
    license: 'AGPL-3.0-only',
    licenseUrl: 'https://www.gnu.org/licenses/agpl-3.0.html',
    fetch: vi.fn(),
  }),
}))

describe('setup-error page', () => {
  it('renders the source and licence notice', () => {
    const w = shallowMount(SetupErrorPage, {
      global: { components: { AppSourceNotice }, stubs: { AppSourceNotice: false } },
    })
    expect(w.find('[data-testid="source-notice"]').exists()).toBe(true)
  })
})

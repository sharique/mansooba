// @vitest-environment happy-dom
// The source and licence notice (AGPL-3.0 section 13) lives in the page footer
// of the signed-in layout. The footer renders at every screen width, unlike the
// sidebar, which is a collapsed drawer below the `lg` breakpoint.
import { shallowMount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import Footer from './Footer.vue'
import AppSourceNotice from '../common/AppSourceNotice.vue'

vi.mock('~/stores/about.store', () => ({
  useAboutStore: () => ({
    version: '1.4.0',
    sourceUrl: 'https://github.com/sharique/mansooba',
    license: 'AGPL-3.0-only',
    licenseUrl: 'https://www.gnu.org/licenses/agpl-3.0.html',
    fetch: vi.fn(),
  }),
}))

describe('Footer', () => {
  const mountIt = () =>
    shallowMount(Footer, { global: { components: { AppSourceNotice }, stubs: { AppSourceNotice: false } } })

  it('is a footer landmark that holds the source and licence notice', () => {
    const w = mountIt()
    expect(w.element.tagName).toBe('FOOTER')
    expect(w.find('[data-testid="source-notice"]').exists()).toBe(true)
  })

  it('links to the source and the licence', () => {
    const w = mountIt()
    const hrefs = w.findAll('a').map(a => a.attributes('href'))
    expect(hrefs).toContain('https://github.com/sharique/mansooba')
    expect(hrefs).toContain('https://www.gnu.org/licenses/agpl-3.0.html')
  })
})

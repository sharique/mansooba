// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AppSourceNotice from './AppSourceNotice.vue'

let storeState = {
  version: '',
  sourceUrl: '',
  license: '',
  licenseUrl: '',
  fetch: vi.fn(),
}

vi.mock('~/stores/about.store', () => ({
  useAboutStore: () => storeState,
}))

describe('AppSourceNotice', () => {
  it('renders a link to the source URL and a link to the licence URL, with the version, when known', () => {
    storeState = {
      version: '1.4.0',
      sourceUrl: 'https://example.com/my-fork',
      license: 'AGPL-3.0-only',
      licenseUrl: 'https://www.gnu.org/licenses/agpl-3.0.html',
      fetch: vi.fn(),
    }

    const w = mount(AppSourceNotice)
    const notice = w.find('[data-testid="source-notice"]')
    expect(notice.exists()).toBe(true)

    const sourceLink = notice.find('a[href="https://example.com/my-fork"]')
    expect(sourceLink.exists()).toBe(true)
    expect(sourceLink.attributes('target')).toBe('_blank')
    expect(sourceLink.attributes('rel')).toBe('noopener noreferrer')

    const licenseLink = notice.find('a[href="https://www.gnu.org/licenses/agpl-3.0.html"]')
    expect(licenseLink.exists()).toBe(true)
    expect(licenseLink.text()).toBe('AGPL-3.0')
    expect(licenseLink.attributes('rel')).toBe('noopener noreferrer')

    expect(notice.text()).toContain('v1.4.0')
  })

  it('omits the version when it is not yet known', () => {
    storeState = {
      version: '',
      sourceUrl: 'https://github.com/sharique/mansooba',
      license: 'AGPL-3.0-only',
      licenseUrl: 'https://www.gnu.org/licenses/agpl-3.0.html',
      fetch: vi.fn(),
    }

    const w = mount(AppSourceNotice)
    expect(w.find('[data-testid="source-notice"]').text()).not.toMatch(/v\d/)
  })
})

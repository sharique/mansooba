import { setActivePinia, createPinia } from 'pinia'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { useAboutStore } from './about.store'

// fetch() must never throw — the source/licence notice must not break the
// page it's shown on.
const mockGetAbout = vi.fn()

vi.mock('~/services/about.service', () => ({
  aboutService: {
    getAbout: () => mockGetAbout(),
  },
}))

const aboutResponse = {
  version: '1.4.0',
  source_url: 'https://example.com/my-fork',
  license: 'AGPL-3.0-only',
  license_url: 'https://www.gnu.org/licenses/agpl-3.0.html',
}

describe('about store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGetAbout.mockReset()
  })

  test('fetch() fills version, source URL, licence and licence URL from the API response', async () => {
    mockGetAbout.mockResolvedValueOnce(aboutResponse)
    const store = useAboutStore()
    await store.fetch()
    expect(store.version).toBe('1.4.0')
    expect(store.sourceUrl).toBe('https://example.com/my-fork')
    expect(store.license).toBe('AGPL-3.0-only')
    expect(store.licenseUrl).toBe('https://www.gnu.org/licenses/agpl-3.0.html')
  })

  test('a failed request leaves the defaults and never throws', async () => {
    mockGetAbout.mockRejectedValueOnce(new Error('network error'))
    const store = useAboutStore()
    await expect(store.fetch()).resolves.not.toThrow()
    expect(store.sourceUrl).toBe('https://github.com/sharique/mansooba')
    expect(store.license).toBe('AGPL-3.0-only')
  })
})

import { setActivePinia, createPinia } from 'pinia'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { useGlobalSettingsStore } from './global-settings.store'

const mockGetSettings = vi.fn()
const mockPatchSettings = vi.fn()

vi.mock('~/services/settings.service', () => ({
  settingsService: {
    getSettings: () => mockGetSettings(),
    patchSettings: (payload: unknown) => mockPatchSettings(payload),
  },
}))

const defaultSettings = {
  organization_name: 'Mansooba',
  date_format: 'YYYY-MM-DD',
  time_format: '24h',
  locale: 'en-US',
  week_start_day: 'monday',
}

describe('global-settings store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGetSettings.mockReset()
    mockPatchSettings.mockReset()
  })

  test('fetch() populates all five settings fields from API response', async () => {
    mockGetSettings.mockResolvedValueOnce(defaultSettings)
    const store = useGlobalSettingsStore()
    await store.fetch()
    expect(store.organization_name).toBe('Mansooba')
    expect(store.date_format).toBe('YYYY-MM-DD')
    expect(store.time_format).toBe('24h')
    expect(store.locale).toBe('en-US')
    expect(store.week_start_day).toBe('monday')
  })

  test('patch() calls API and updates state on success', async () => {
    const updated = { ...defaultSettings, organization_name: 'Acme Corp' }
    mockPatchSettings.mockResolvedValueOnce(updated)
    const store = useGlobalSettingsStore()
    await store.patch({ organization_name: 'Acme Corp' })
    expect(mockPatchSettings).toHaveBeenCalledWith({ organization_name: 'Acme Corp' })
    expect(store.organization_name).toBe('Acme Corp')
  })
})

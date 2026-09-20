// @vitest-environment happy-dom
// T018 (013-demo-instance-banner): admin can toggle the demo banner and
// edit its message from the global settings form.
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import GlobalSettingsForm from './GlobalSettingsForm.vue'

const mockPatch = vi.fn()

function storeStub() {
  return {
    loaded: true,
    organization_name: 'Mansooba',
    date_format: 'YYYY-MM-DD',
    time_format: '24h',
    locale: 'en-US',
    week_start_day: 'monday',
    system_log_retention_days: '90',
    demo_banner_enabled: 'false',
    demo_banner_message: 'This is a demo instance. Data may be reset at any time.',
    patch: mockPatch,
  }
}

vi.mock('~/stores/global-settings.store', () => ({
  useGlobalSettingsStore: storeStub,
}))
vi.stubGlobal('useGlobalSettingsStore', storeStub)
vi.stubGlobal('useToast', () => ({ showSuccess: vi.fn(), showError: vi.fn() }))

describe('GlobalSettingsForm — demo banner fields', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockPatch.mockReset()
    mockPatch.mockResolvedValue(undefined)
  })

  test('submitting with the banner enabled and a custom message calls store.patch with both fields', async () => {
    const w = mount(GlobalSettingsForm)

    await w.find('[data-testid="demo-banner-enabled"]').setValue(true)
    await w.find('[data-testid="demo-banner-message"]').setValue('Custom text')
    await w.find('form').trigger('submit')

    expect(mockPatch).toHaveBeenCalledWith(
      expect.objectContaining({
        demo_banner_enabled: 'true',
        demo_banner_message: 'Custom text',
      }),
    )
  })

  test('submitting with the banner enabled and the message cleared shows an inline error and does not call store.patch', async () => {
    const w = mount(GlobalSettingsForm)

    await w.find('[data-testid="demo-banner-enabled"]').setValue(true)
    await w.find('[data-testid="demo-banner-message"]').setValue('')
    await w.find('form').trigger('submit')

    expect(w.find('[data-testid="demo-banner-message-error"]').exists()).toBe(true)
    expect(mockPatch).not.toHaveBeenCalled()
  })
})

// @vitest-environment happy-dom
// T007 (013-demo-instance-banner): DemoBanner renders the current
// demo-instance message when enabled, and nothing when disabled.
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DemoBanner from './DemoBanner.vue'

describe('DemoBanner', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('renders the banner with the current message when enabled', () => {
    vi.stubGlobal('useSetupStore', () => ({
      demoBannerEnabled: true,
      demoBannerMessage: 'Custom text',
    }))

    const w = mount(DemoBanner)

    const banner = w.find('[data-testid="demo-banner"]')
    expect(banner.exists()).toBe(true)
    expect(banner.text()).toBe('Custom text')
  })

  it('renders nothing when disabled', () => {
    vi.stubGlobal('useSetupStore', () => ({
      demoBannerEnabled: false,
      demoBannerMessage: '',
    }))

    const w = mount(DemoBanner)

    expect(w.find('[data-testid="demo-banner"]').exists()).toBe(false)
  })
})

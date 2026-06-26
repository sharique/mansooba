import { setActivePinia, createPinia } from 'pinia'
import { beforeEach, describe, expect, test, vi } from 'vitest'

// navigateTo is a Nuxt auto-import unavailable in unit tests.
;(globalThis as Record<string, unknown>).navigateTo = vi.fn()

const mockSeedData = vi.fn()

vi.mock('~/services/setup.service', () => ({
  setupService: {
    getStatus: vi.fn().mockResolvedValue({ setup_required: true }),
    createAdmin: vi.fn(),
    createUser: vi.fn(),
    createProject: vi.fn(),
    seedData: () => mockSeedData(),
  },
}))

// Prevent real navigation in tests.
vi.mock('#app', () => ({
  useNuxtApp: vi.fn(),
  navigateTo: vi.fn(),
  defineNuxtRouteMiddleware: vi.fn(),
}))

import { useSetupStore } from './setup.store'

describe('setup store — seed actions', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockSeedData.mockReset()
  })

  test('completeSampleData sets seedImported=true and advances to step 5 on success', async () => {
    mockSeedData.mockResolvedValue({
      skipped: false,
      project_key: 'DEMO',
      project_name: 'Mansooba Demo',
    })

    const store = useSetupStore()
    store.currentStep = 4

    await store.completeSampleData()

    expect(store.seedImported).toBe(true)
    expect(store.seedProjectKey).toBe('DEMO')
    expect(store.currentStep).toBe(5)
  })

  test('completeSampleData sets seedImported=false when skipped=true', async () => {
    mockSeedData.mockResolvedValue({
      skipped: true,
      project_key: 'DEMO',
      project_name: 'Mansooba Demo',
    })

    const store = useSetupStore()
    await store.completeSampleData()

    expect(store.seedImported).toBe(false)
    expect(store.seedProjectKey).toBe('DEMO')
    expect(store.currentStep).toBe(5)
  })

  test('skipSampleData sets seedImported=false and advances to step 5', () => {
    const store = useSetupStore()
    store.currentStep = 4

    store.skipSampleData()

    expect(store.seedImported).toBe(false)
    expect(store.seedProjectKey).toBeNull()
    expect(store.currentStep).toBe(5)
  })

  test('finish resets seed fields to null', () => {
    const store = useSetupStore()
    store.seedImported = true
    store.seedProjectKey = 'DEMO'
    store.currentStep = 5

    store.finish()

    expect(store.seedImported).toBeNull()
    expect(store.seedProjectKey).toBeNull()
    expect(store.currentStep).toBe(0)
  })

  test('summaryItems includes seed entry when seedImported is not null', () => {
    const store = useSetupStore()
    store.seedImported = true
    store.seedProjectKey = 'DEMO'

    const items = store.summaryItems
    const seedItem = items.find((i) => i.label === 'Sample data')
    expect(seedItem).toBeDefined()
    expect(seedItem?.value).toContain('DEMO')
  })

  test('summaryItems shows Skipped when seedImported=false', () => {
    const store = useSetupStore()
    store.seedImported = false

    const items = store.summaryItems
    const seedItem = items.find((i) => i.label === 'Sample data')
    expect(seedItem?.value).toBe('Skipped')
  })

  test('summaryItems omits seed entry when seedImported is null', () => {
    const store = useSetupStore()
    // seedImported defaults to null

    const items = store.summaryItems
    const seedItem = items.find((i) => i.label === 'Sample data')
    expect(seedItem).toBeUndefined()
  })
})

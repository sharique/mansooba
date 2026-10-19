// @vitest-environment happy-dom
import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, it, expect, vi } from 'vitest'
import BoardPage from './board.vue'

vi.mock('~/services/board.service', () => ({
  boardService: { getBoard: vi.fn().mockResolvedValue({ columns: [] }) },
}))

vi.stubGlobal('useRoute', () => ({ params: { key: 'TEST' } }))
vi.stubGlobal('useIssuesStore', () => ({ update: vi.fn() }))
vi.stubGlobal('useToast', () => ({ showSuccess: vi.fn(), showError: vi.fn() }))

describe('board page', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('does not render a burndown chart (it lives on the Reports page)', async () => {
    const w = shallowMount(BoardPage)
    await flushPromises()
    expect(w.find('section.mb-6').exists()).toBe(false)
    expect(w.find('chartsburndownchart').exists()).toBe(false)
  })

  it('shows "Backlog →" navigation link', async () => {
    const w = shallowMount(BoardPage)
    await flushPromises()
    expect(w.text()).toContain('Backlog →')
  })

  it('links to the Sprints page next to the Backlog link', async () => {
    const w = shallowMount(BoardPage)
    await flushPromises()
    expect(w.find('nuxtlink[to="/projects/TEST/sprints"]').exists()).toBe(true)
    expect(w.text()).toContain('Sprints →')
  })
})

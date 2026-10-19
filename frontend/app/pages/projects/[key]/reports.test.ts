// @vitest-environment happy-dom
import { flushPromises, shallowMount } from '@vue/test-utils'
import { describe, it, expect, vi } from 'vitest'
import ReportsPage from './reports.vue'

vi.mock('~/services/reports.service', () => ({
  reportsService: { getVelocity: vi.fn().mockResolvedValue([]) },
}))
vi.stubGlobal('useRoute', () => ({ params: { key: 'TEST' } }))

describe('reports page', () => {
  it('renders the burndown section for the project and the velocity report', async () => {
    const w = shallowMount(ReportsPage)
    await flushPromises()
    expect(w.find('reportsburndownsection').attributes('project-key')).toBe('TEST')
    expect(w.text()).toContain('No completed sprints yet')
    expect(w.find('section .card h2').exists()).toBe(true)
    expect(w.text()).toContain('Sprint velocity')
  })
})

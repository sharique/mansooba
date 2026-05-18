// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, it, expect, vi } from 'vitest'
import BurndownChart from './BurndownChart.vue'
import type { BurndownData } from '~/types/domain.types'

// canvas doesn't work in happy-dom; replace Line with a plain stub
vi.mock('vue-chartjs', () => ({
  Line: { template: '<canvas data-testid="chart" />', props: ['data', 'options'] },
}))

const burndown: BurndownData = {
  sprint_id: 'sprint-1',
  sprint_name: 'Sprint 1',
  start_date: '2026-05-01',
  end_date: '2026-05-03',
  total_points: 10,
  data: [
    { date: '2026-05-01', remaining_points: 10 },
    { date: '2026-05-02', remaining_points: 5 },
    { date: '2026-05-03', remaining_points: 0 },
  ],
}

describe('BurndownChart', () => {
  it('mounts without error', () => {
    expect(() => mount(BurndownChart, { props: { data: burndown } })).not.toThrow()
  })

  it('renders a chart element', () => {
    const w = mount(BurndownChart, { props: { data: burndown } })
    expect(w.find('[data-testid="chart"]').exists()).toBe(true)
  })
})

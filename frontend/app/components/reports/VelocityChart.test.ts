// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, it, expect } from 'vitest'
import VelocityChart from './VelocityChart.vue'
import type { VelocityDataPoint } from '~/types/domain.types'

const sample: VelocityDataPoint[] = [
  { sprint_id: 1, sprint_name: 'Sprint 1', committed: 10, completed: 7 },
  { sprint_id: 2, sprint_name: 'Sprint 2', committed: 8, completed: 8 },
]

describe('VelocityChart', () => {
  it('shows "No completed sprints yet." when data is empty', () => {
    const w = mount(VelocityChart, { props: { data: [] } })
    expect(w.text()).toContain('No completed sprints yet.')
  })

  it('does not show the empty state when data is present', () => {
    const w = mount(VelocityChart, { props: { data: sample } })
    expect(w.text()).not.toContain('No completed sprints yet.')
  })

  it('renders a bar group for each data point', () => {
    const w = mount(VelocityChart, { props: { data: sample } })
    // Each sprint name should appear in the label row.
    expect(w.text()).toContain('Sprint 1')
    expect(w.text()).toContain('Sprint 2')
  })

  it('renders committed and completed values for each sprint', () => {
    const w = mount(VelocityChart, { props: { data: sample } })
    // The numeric values appear as bar labels.
    expect(w.text()).toContain('10') // committed Sprint 1
    expect(w.text()).toContain('7')  // completed Sprint 1
    expect(w.text()).toContain('8')  // committed/completed Sprint 2
  })

  it('displays the legend with Committed and Completed labels', () => {
    const w = mount(VelocityChart, { props: { data: sample } })
    expect(w.text()).toContain('Committed')
    expect(w.text()).toContain('Completed')
  })

  it('renders committed bars with bg-neutral class', () => {
    const w = mount(VelocityChart, { props: { data: sample } })
    const committedBars = w.findAll('.bg-neutral')
    // One committed bar per sprint (legend swatch + bar)
    expect(committedBars.length).toBeGreaterThanOrEqual(sample.length)
  })

  it('renders completed bars with bg-success class', () => {
    const w = mount(VelocityChart, { props: { data: sample } })
    const completedBars = w.findAll('.bg-success')
    expect(completedBars.length).toBeGreaterThanOrEqual(sample.length)
  })

  it('applies proportional height style to bars', () => {
    const single: VelocityDataPoint[] = [
      { sprint_id: 1, sprint_name: 'Sprint 1', committed: 10, completed: 5 },
    ]
    const w = mount(VelocityChart, { props: { data: single } })
    // committed bar should be 100% (it's the max), completed should be 50%
    const committedBar = w.find('.bg-neutral.rounded-t')
    const completedBar = w.find('.bg-success.rounded-t')
    expect(committedBar.attributes('style')).toContain('height: 100%')
    expect(completedBar.attributes('style')).toContain('height: 50%')
  })
})

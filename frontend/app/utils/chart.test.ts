import { describe, it, expect } from 'vitest'
import { toBurndownChartData } from '~/utils/chart'
import type { BurndownData } from '~/types/domain.types'

const makeBurndown = (overrides: Partial<BurndownData> = {}): BurndownData => ({
  sprint_id: 'sprint-1',
  sprint_name: 'Sprint 1',
  start_date: '2026-05-01',
  end_date: '2026-05-03',
  total_points: 10,
  data: [
    { date: '2026-05-01', remaining_points: 10 },
    { date: '2026-05-02', remaining_points: 6 },
    { date: '2026-05-03', remaining_points: 2 },
  ],
  ...overrides,
})

describe('toBurndownChartData', () => {
  it('labels match dates', () => {
    const result = toBurndownChartData(makeBurndown())
    expect(result.labels).toEqual(['2026-05-01', '2026-05-02', '2026-05-03'])
  })

  it('actual dataset contains remaining_points values', () => {
    const result = toBurndownChartData(makeBurndown())
    expect(result.datasets[0]!.data).toEqual([10, 6, 2])
  })

  it('ideal dataset starts at total_points and ends at 0', () => {
    const result = toBurndownChartData(makeBurndown())
    const ideal = result.datasets[1]!.data as number[]
    expect(ideal[0]).toBe(10)
    expect(ideal[ideal.length - 1]).toBe(0)
  })

  it('ideal dataset is linear', () => {
    const result = toBurndownChartData(makeBurndown())
    const ideal = result.datasets[1]!.data as number[]
    expect(ideal).toEqual([10, 5, 0])
  })

  it('handles single-day sprint without dividing by zero', () => {
    const singleDay = makeBurndown({
      data: [{ date: '2026-05-01', remaining_points: 10 }],
    })
    expect(() => toBurndownChartData(singleDay)).not.toThrow()
    const result = toBurndownChartData(singleDay)
    expect(result.datasets[1]!.data).toEqual([10])
  })

  it('produces two datasets: actual and ideal', () => {
    const result = toBurndownChartData(makeBurndown())
    expect(result.datasets).toHaveLength(2)
    expect(result.datasets[0]!.label).toBe('Remaining Points')
    expect(result.datasets[1]!.label).toBe('Ideal')
  })
})

import type { ChartData, ChartDataset } from 'chart.js'
import type { BurndownData } from '~/types/domain.types'

/**
 * Transforms a BurndownData API response into a Chart.js line chart dataset.
 * Produces two lines:
 *   - "Remaining Points": actual story points remaining per day
 *   - "Ideal": a linear guide from total_points to 0 over the sprint duration
 */
export function toBurndownChartData(burndown: BurndownData): ChartData<'line'> {
  const labels = burndown.data.map(p => p.date)
  const actual = burndown.data.map(p => p.remaining_points)

  const total = burndown.total_points
  const days = burndown.data.length
  const ideal = burndown.data.map((_, i) =>
    days <= 1 ? total : Math.round(total - (total / (days - 1)) * i),
  )

  const actualDataset: ChartDataset<'line'> = {
    label: 'Remaining Points',
    data: actual,
    borderColor: 'rgb(59, 130, 246)',
    backgroundColor: 'rgba(59, 130, 246, 0.1)',
    tension: 0.1,
    pointRadius: 3,
  }

  const idealDataset: ChartDataset<'line'> = {
    label: 'Ideal',
    data: ideal,
    borderColor: 'rgb(156, 163, 175)',
    borderDash: [5, 5],
    backgroundColor: 'transparent',
    tension: 0,
    pointRadius: 0,
  }

  return { labels, datasets: [actualDataset, idealDataset] }
}

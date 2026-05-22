import { beforeEach, describe, expect, test, vi } from 'vitest'
import type { VelocityDataPoint } from '~/types/domain.types'

const mockApi = vi.fn()
vi.stubGlobal('useNuxtApp', () => ({ $api: mockApi }))

describe('reportsService', () => {
  beforeEach(() => mockApi.mockReset())

  test('getVelocity calls GET /projects/:key/velocity', async () => {
    const { reportsService } = await import('./reports.service')
    mockApi.mockResolvedValueOnce([])
    await reportsService.getVelocity('PROJ')
    expect(mockApi).toHaveBeenCalledWith('/projects/PROJ/velocity')
  })

  test('getVelocity returns the array of VelocityDataPoint from the API', async () => {
    const { reportsService } = await import('./reports.service')
    const fixture: VelocityDataPoint[] = [
      { sprint_id: 1, sprint_name: 'Sprint 1', committed: 10, completed: 7 },
      { sprint_id: 2, sprint_name: 'Sprint 2', committed: 8, completed: 8 },
    ]
    mockApi.mockResolvedValueOnce(fixture)
    const result = await reportsService.getVelocity('PROJ')
    expect(result).toHaveLength(2)
    expect(result[0].sprint_name).toBe('Sprint 1')
    expect(result[1].completed).toBe(8)
  })

  test('getVelocity propagates API errors', async () => {
    const { reportsService } = await import('./reports.service')
    mockApi.mockRejectedValueOnce(new Error('network error'))
    await expect(reportsService.getVelocity('PROJ')).rejects.toThrow('network error')
  })
})

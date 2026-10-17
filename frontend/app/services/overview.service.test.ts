import { beforeEach, describe, expect, test, vi } from 'vitest'

const mockApi = vi.fn()
vi.stubGlobal('useNuxtApp', () => ({ $api: mockApi }))

describe('overviewService', () => {
  beforeEach(() => mockApi.mockReset())

  test('getSprint calls GET /projects/:key/overview/sprint and unwraps the sprint', async () => {
    const { overviewService } = await import('./overview.service')
    mockApi.mockResolvedValueOnce({ sprint: { id: 3, name: 'Sprint 3' } })
    const sprint = await overviewService.getSprint('PROJ')
    expect(mockApi).toHaveBeenCalledWith('/projects/PROJ/overview/sprint')
    expect(sprint?.name).toBe('Sprint 3')
  })

  test('getSprint returns null when no sprint is active', async () => {
    const { overviewService } = await import('./overview.service')
    mockApi.mockResolvedValueOnce({ sprint: null })
    expect(await overviewService.getSprint('PROJ')).toBeNull()
  })

  test('getTasks calls GET /projects/:key/overview/tasks', async () => {
    const { overviewService } = await import('./overview.service')
    mockApi.mockResolvedValueOnce({ sprint_active: true, items: [] })
    const res = await overviewService.getTasks('PROJ')
    expect(mockApi).toHaveBeenCalledWith('/projects/PROJ/overview/tasks')
    expect(res.sprint_active).toBe(true)
  })

  test('getActivity passes the scope as a query parameter', async () => {
    const { overviewService } = await import('./overview.service')
    mockApi.mockResolvedValue({ sprint_active: false, items: [] })
    await overviewService.getActivity('PROJ', 'sprint')
    expect(mockApi).toHaveBeenLastCalledWith('/projects/PROJ/overview/activity', { query: { scope: 'sprint' } })
    await overviewService.getActivity('PROJ', 'all')
    expect(mockApi).toHaveBeenLastCalledWith('/projects/PROJ/overview/activity', { query: { scope: 'all' } })
  })

  test('errors propagate', async () => {
    const { overviewService } = await import('./overview.service')
    mockApi.mockRejectedValueOnce(new Error('403'))
    await expect(overviewService.getTasks('PROJ')).rejects.toThrow('403')
  })
})

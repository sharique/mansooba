import { beforeEach, describe, expect, test, vi } from 'vitest'

const mockApi = vi.fn()
vi.stubGlobal('useNuxtApp', () => ({ $api: mockApi }))

describe('adminService', () => {
  beforeEach(() => mockApi.mockReset())

  test('listProjectsWithoutAdmin calls GET /admin/projects/without-admin', async () => {
    const { adminService } = await import('./admin.service')
    mockApi.mockResolvedValueOnce([{ id: 1, key: 'old', name: 'Old', member_count: 2 }])
    const rows = await adminService.listProjectsWithoutAdmin()
    expect(mockApi).toHaveBeenCalledWith('/admin/projects/without-admin')
    expect(rows).toHaveLength(1)
  })

  test('listProjectsWithoutAdmin returns [] for a null body', async () => {
    const { adminService } = await import('./admin.service')
    mockApi.mockResolvedValueOnce(null)
    expect(await adminService.listProjectsWithoutAdmin()).toEqual([])
  })

  test('claimProjectAdmin calls POST /admin/projects/:key/claim-admin', async () => {
    const { adminService } = await import('./admin.service')
    mockApi.mockResolvedValueOnce({})
    await adminService.claimProjectAdmin('old')
    expect(mockApi).toHaveBeenCalledWith('/admin/projects/old/claim-admin', { method: 'POST' })
  })
})

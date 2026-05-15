import { beforeEach, describe, expect, test, vi } from 'vitest'

const mockApi = vi.fn()
vi.stubGlobal('useNuxtApp', () => ({ $api: mockApi }))

describe('projectsService', () => {
  beforeEach(() => mockApi.mockReset())

  test('list calls GET /projects', async () => {
    const { projectsService } = await import('./projects.service')
    mockApi.mockResolvedValueOnce([])
    await projectsService.list()
    expect(mockApi).toHaveBeenCalledWith('/projects')
  })

  test('create calls POST /projects with body', async () => {
    const { projectsService } = await import('./projects.service')
    mockApi.mockResolvedValueOnce({ id: 1, key: 'PROJ', name: 'P', description: '', ownerId: 1 })
    await projectsService.create({ name: 'P', key: 'PROJ' })
    expect(mockApi).toHaveBeenCalledWith('/projects', { method: 'POST', body: { name: 'P', key: 'PROJ' } })
  })

  test('removeMember calls DELETE /projects/:key/members/:id', async () => {
    const { projectsService } = await import('./projects.service')
    mockApi.mockResolvedValueOnce(undefined)
    await projectsService.removeMember('PROJ', 42)
    expect(mockApi).toHaveBeenCalledWith('/projects/PROJ/members/42', { method: 'DELETE' })
  })
})

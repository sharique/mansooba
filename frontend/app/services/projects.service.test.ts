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

  test('get maps my_role and permissions to camelCase', async () => {
    const { projectsService } = await import('./projects.service')
    mockApi.mockResolvedValueOnce({ id: 1, key: 'p', name: 'P', description: '', owner_id: 3, my_role: 'viewer', permissions: ['project.view'] })
    const p = await projectsService.get('p')
    expect(p.myRole).toBe('viewer')
    expect(p.permissions).toEqual(['project.view'])
    expect(p.ownerId).toBe(3)
  })

  test('list does not invent permissions', async () => {
    const { projectsService } = await import('./projects.service')
    mockApi.mockResolvedValueOnce([{ id: 1, key: 'p', name: 'P', description: '', owner_id: 3 }])
    const [p] = await projectsService.list()
    expect(p!.permissions).toBeUndefined()
  })

  test('changeMemberRole calls PATCH /projects/:key/members/:id with the role', async () => {
    const { projectsService } = await import('./projects.service')
    mockApi.mockResolvedValueOnce({})
    await projectsService.changeMemberRole('PROJ', 4, 'viewer')
    expect(mockApi).toHaveBeenCalledWith('/projects/PROJ/members/4', { method: 'PATCH', body: { role: 'viewer' } })
  })

  test('deleteProject calls DELETE with the confirm_name body', async () => {
    const { projectsService } = await import('./projects.service')
    mockApi.mockResolvedValueOnce(undefined)
    await projectsService.deleteProject('PROJ', 'My Project')
    expect(mockApi).toHaveBeenCalledWith('/projects/PROJ', { method: 'DELETE', body: { confirm_name: 'My Project' } })
  })
})

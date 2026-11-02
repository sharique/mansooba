import { setActivePinia, createPinia } from 'pinia'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { useAdminProjectsStore } from './admin-projects.store'

const list = vi.fn()
const claim = vi.fn()
vi.mock('~/services/admin.service', () => ({
  adminService: { listProjectsWithoutAdmin: () => list(), claimProjectAdmin: (k: string) => claim(k) },
}))

const row = (key: string) => ({ id: 1, key, name: key.toUpperCase(), member_count: 2 })

describe('admin projects store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    list.mockReset()
    claim.mockReset()
  })

  test('fetch fills the list of projects without an admin', async () => {
    list.mockResolvedValueOnce([row('old')])
    const s = useAdminProjectsStore()
    await s.fetch()
    expect(s.projects).toHaveLength(1)
    expect(s.error).toBeNull()
    expect(s.loading).toBe(false)
  })

  test('a failed fetch records the server message', async () => {
    list.mockRejectedValueOnce({ data: { message: 'nope' } })
    const s = useAdminProjectsStore()
    await s.fetch()
    expect(s.error).toBe('nope')
    expect(s.projects).toEqual([])
  })

  test('claim removes the project from the list', async () => {
    list.mockResolvedValueOnce([row('old'), row('other')])
    claim.mockResolvedValueOnce({})
    const s = useAdminProjectsStore()
    await s.fetch()
    await s.claim('old')
    expect(claim).toHaveBeenCalledWith('old')
    expect(s.projects.map(p => p.key)).toEqual(['other'])
  })

  test('claim surfaces the server message when another admin exists (409)', async () => {
    list.mockResolvedValueOnce([row('old')])
    claim.mockRejectedValueOnce({ status: 409, data: { message: 'This project already has an active admin' } })
    const s = useAdminProjectsStore()
    await s.fetch()
    await expect(s.claim('old')).rejects.toThrow('This project already has an active admin')
    expect(s.projects).toHaveLength(1)
  })
})

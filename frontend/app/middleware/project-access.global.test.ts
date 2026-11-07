// @vitest-environment happy-dom
import { beforeEach, describe, expect, it, vi } from 'vitest'

const navigateTo = vi.fn((r: unknown) => r)
const auth = { isAuthenticated: true }
const store: { current: { key: string } | null, fetchOne: ReturnType<typeof vi.fn> } = { current: null, fetchOne: vi.fn() }

vi.stubGlobal('defineNuxtRouteMiddleware', (fn: unknown) => fn)
vi.stubGlobal('navigateTo', navigateTo)
vi.mock('~/stores/auth.store', () => ({ useAuthStore: () => auth }))
vi.mock('~/stores/projects.store', () => ({ useProjectsStore: () => store }))

const run = async (path: string) => {
  const mod = await import('./project-access.global')
  return (mod.default as unknown as (to: { path: string }) => Promise<unknown>)({ path })
}

describe('project-access middleware', () => {
  beforeEach(() => {
    navigateTo.mockClear()
    store.current = null
    store.fetchOne = vi.fn().mockResolvedValue({})
    auth.isAuthenticated = true
  })

  it('sends a non-member (404) to the no-access page with the project key', async () => {
    store.fetchOne.mockRejectedValue({ status: 404 })
    await run('/projects/Sharique/board')
    expect(navigateTo).toHaveBeenCalledWith({ path: '/no-access', query: { project: 'sharique' } }, { replace: true })
  })

  it('lets a member through', async () => {
    await run('/projects/demo')
    expect(navigateTo).not.toHaveBeenCalled()
  })

  it('ignores pages outside a project and signed-out visitors', async () => {
    await run('/settings')
    auth.isAuthenticated = false
    await run('/projects/demo')
    expect(store.fetchOne).not.toHaveBeenCalled()
  })

  it('lets the page handle other failures itself', async () => {
    store.fetchOne.mockRejectedValue({ status: 500 })
    await run('/projects/demo')
    expect(navigateTo).not.toHaveBeenCalled()
  })

  it('does not refetch the project that is already loaded', async () => {
    store.current = { key: 'demo' }
    await run('/projects/demo/board')
    expect(store.fetchOne).not.toHaveBeenCalled()
  })
})

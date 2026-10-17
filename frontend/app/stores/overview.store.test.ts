import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useOverviewStore } from '~/stores/overview.store'
import { overviewService } from '~/services/overview.service'

vi.mock('~/services/overview.service', () => ({
  overviewService: { getSprint: vi.fn(), getTasks: vi.fn(), getActivity: vi.fn() },
}))

const svc = vi.mocked(overviewService)
const sprint = { id: 1, name: 'Sprint 1', is_overdue: false } as never
const tasks = { sprint_active: true, items: [{ key: 'P-1' }] } as never
const activity = (items: unknown[] = []) => ({ sprint_active: true, items }) as never

describe('useOverviewStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.resetAllMocks()
  })

  it('loads each section independently', async () => {
    svc.getSprint.mockResolvedValue(sprint)
    svc.getTasks.mockResolvedValue(tasks)
    svc.getActivity.mockResolvedValue(activity([{ id: 1 }]))
    const store = useOverviewStore()

    await store.fetchSprint('P')
    expect(store.sprint.data?.name).toBe('Sprint 1')
    expect(store.tasks.data).toBeNull()
    expect(store.activity.data).toBeNull()

    await store.fetchTasks('P')
    await store.fetchActivity('P')
    expect(store.tasks.data?.items).toHaveLength(1)
    expect(store.activity.data?.items).toHaveLength(1)
  })

  it('a failing section keeps its own error and leaves the others alone', async () => {
    svc.getSprint.mockResolvedValue(sprint)
    svc.getTasks.mockRejectedValue({ data: { message: 'forbidden' } })
    const store = useOverviewStore()
    await store.fetchSprint('P')
    await store.fetchTasks('P')

    expect(store.tasks.error).toBe('forbidden')
    expect(store.tasks.data).toBeNull()
    expect(store.tasks.loading).toBe(false)
    expect(store.sprint.error).toBeNull()
    expect(store.sprint.data?.name).toBe('Sprint 1')
  })

  it('sets loading while a request is in flight and clears it after', async () => {
    let resolve!: (v: never) => void
    svc.getSprint.mockReturnValue(new Promise(r => { resolve = r as never }))
    const store = useOverviewStore()
    const p = store.fetchSprint('P')
    expect(store.sprint.loading).toBe(true)
    resolve(sprint)
    await p
    expect(store.sprint.loading).toBe(false)
  })

  it('retry after an error clears it on success', async () => {
    svc.getTasks.mockRejectedValueOnce(new Error('boom')).mockResolvedValueOnce(tasks)
    const store = useOverviewStore()
    await store.fetchTasks('P')
    expect(store.tasks.error).toBe('boom')
    await store.fetchTasks('P')
    expect(store.tasks.error).toBeNull()
    expect(store.tasks.data?.items).toHaveLength(1)
  })

  it('activity starts on "all" and fetchActivity stores the chosen scope', async () => {
    svc.getActivity.mockResolvedValue(activity())
    const store = useOverviewStore()
    expect(store.scope).toBe('all')
    await store.fetchActivity('P', 'sprint')
    expect(store.scope).toBe('sprint')
    expect(svc.getActivity).toHaveBeenLastCalledWith('P', 'sprint')
    await store.fetchActivity('P')
    expect(svc.getActivity).toHaveBeenLastCalledWith('P', 'sprint') // keeps the scope when none is given
    expect(svc.getSprint).not.toHaveBeenCalled()
    expect(svc.getTasks).not.toHaveBeenCalled()
  })

  it('ignores a slow answer for a scope the person has already left', async () => {
    let resolveAll!: (v: never) => void
    svc.getActivity
      .mockReturnValueOnce(new Promise(r => { resolveAll = r as never }))
      .mockResolvedValueOnce(activity([{ id: 2, kind: 'sprint-only' }]))
    const store = useOverviewStore()
    const slow = store.fetchActivity('P', 'all')
    await store.fetchActivity('P', 'sprint')
    resolveAll(activity([{ id: 1, kind: 'all-late' }]))
    await slow
    expect(store.scope).toBe('sprint')
    expect(store.activity.data?.items).toEqual([{ id: 2, kind: 'sprint-only' }])
    expect(store.activity.loading).toBe(false)
  })

  it('reset empties every section and returns to "all"', async () => {
    svc.getSprint.mockResolvedValue(sprint)
    svc.getActivity.mockResolvedValue(activity())
    const store = useOverviewStore()
    await store.fetchSprint('P')
    await store.fetchActivity('P', 'sprint')
    store.reset()
    expect(store.sprint.data).toBeNull()
    expect(store.activity.data).toBeNull()
    expect(store.scope).toBe('all')
  })
})

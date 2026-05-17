import { setActivePinia, createPinia } from 'pinia'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { useSprintsStore } from './sprints.store'
import type { Sprint, BurndownData } from '~/types/domain.types'

const mockList = vi.fn()
const mockGet = vi.fn()
const mockCreate = vi.fn()
const mockUpdate = vi.fn()
const mockDelete = vi.fn()
const mockStart = vi.fn()
const mockComplete = vi.fn()
const mockBurndown = vi.fn()

vi.mock('~/services/sprints.service', () => ({
  sprintsService: {
    list: (key: string) => mockList(key),
    get: (key: string, id: string) => mockGet(key, id),
    create: (key: string, payload: unknown) => mockCreate(key, payload),
    update: (key: string, id: string, payload: unknown) => mockUpdate(key, id, payload),
    delete: (key: string, id: string) => mockDelete(key, id),
    start: (key: string, id: string) => mockStart(key, id),
    complete: (key: string, id: string, payload: unknown) => mockComplete(key, id, payload),
    burndown: (key: string, id: string) => mockBurndown(key, id),
  },
}))

const sprint: Sprint = {
  id: 'sprint-1',
  project_id: 'proj-1',
  name: 'Sprint 1',
  goal: '',
  status: 'planning',
  start_date: null,
  end_date: null,
  created_at: '2026-05-20T00:00:00Z',
  updated_at: '2026-05-20T00:00:00Z',
}

const activeSprint: Sprint = { ...sprint, id: 'sprint-active', status: 'active' }

const burndown: BurndownData = {
  sprint_id: 'sprint-active',
  sprint_name: 'Sprint Active',
  start_date: '2026-05-01',
  end_date: '2026-05-14',
  total_points: 20,
  data: [{ date: '2026-05-01', remaining_points: 20 }],
}

describe('useSprintsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockList.mockReset()
    mockGet.mockReset()
    mockCreate.mockReset()
    mockUpdate.mockReset()
    mockDelete.mockReset()
    mockStart.mockReset()
    mockComplete.mockReset()
    mockBurndown.mockReset()
  })

  test('fetchSprints populates sprints list', async () => {
    mockList.mockResolvedValueOnce([sprint])
    const store = useSprintsStore()
    await store.fetchSprints('TEST')
    expect(store.sprints).toHaveLength(1)
    const sprint0 = store.sprints[0]
    expect(sprint0).toBeDefined()
    expect(sprint0!.name).toBe('Sprint 1')
  })

  test('fetchSprints sets error on failure', async () => {
    mockList.mockRejectedValueOnce({ message: 'Network error' })
    const store = useSprintsStore()
    await store.fetchSprints('TEST')
    expect(store.error).toBe('Network error')
    expect(store.loading).toBe(false)
  })

  test('activeSprint returns null when no sprint is active', async () => {
    mockList.mockResolvedValueOnce([sprint])
    const store = useSprintsStore()
    await store.fetchSprints('TEST')
    expect(store.activeSprint).toBeNull()
  })

  test('activeSprint returns the active sprint', async () => {
    mockList.mockResolvedValueOnce([sprint, activeSprint])
    const store = useSprintsStore()
    await store.fetchSprints('TEST')
    expect(store.activeSprint?.id).toBe('sprint-active')
  })

  test('openSprints excludes completed sprints', async () => {
    const completed: Sprint = { ...sprint, id: 'sprint-done', status: 'completed' }
    mockList.mockResolvedValueOnce([sprint, activeSprint, completed])
    const store = useSprintsStore()
    await store.fetchSprints('TEST')
    expect(store.openSprints).toHaveLength(2)
    expect(store.openSprints.every(s => s.status !== 'completed')).toBe(true)
  })

  test('getSprint updates the sprint in the list', async () => {
    const updated: Sprint = { ...sprint, name: 'Sprint 1 Updated' }
    mockGet.mockResolvedValueOnce(updated)
    const store = useSprintsStore()
    store.sprints = [sprint]
    const result = await store.getSprint('TEST', sprint.id)
    expect(result.name).toBe('Sprint 1 Updated')
    const sprint0 = store.sprints[0]
    expect(sprint0).toBeDefined()
    expect(sprint0!.name).toBe('Sprint 1 Updated')
  })

  test('getSprint sets error on failure', async () => {
    mockGet.mockRejectedValueOnce({ message: 'Not found' })
    const store = useSprintsStore()
    await expect(store.getSprint('TEST', 'bad-id')).rejects.toBeDefined()
    expect(store.error).toBe('Not found')
  })

  test('createSprint appends to list', async () => {
    mockCreate.mockResolvedValueOnce(sprint)
    const store = useSprintsStore()
    const result = await store.createSprint('TEST', { name: 'Sprint 1' })
    expect(store.sprints).toHaveLength(1)
    expect(result.id).toBe('sprint-1')
  })

  test('createSprint sets error on failure', async () => {
    mockCreate.mockRejectedValueOnce({ message: 'Forbidden' })
    const store = useSprintsStore()
    await expect(store.createSprint('TEST', { name: 'Sprint 1' })).rejects.toBeDefined()
    expect(store.error).toBe('Forbidden')
  })

  test('startSprint updates the sprint status in the list', async () => {
    const started: Sprint = { ...sprint, status: 'active' }
    mockStart.mockResolvedValueOnce(started)
    const store = useSprintsStore()
    store.sprints = [sprint]
    await store.startSprint('TEST', sprint.id)
    const sprint0 = store.sprints[0]
    expect(sprint0).toBeDefined()
    expect(sprint0!.status).toBe('active')
  })

  test('completeSprint calls service with next_sprint_id and updates list', async () => {
    const completed: Sprint = { ...activeSprint, status: 'completed' }
    mockComplete.mockResolvedValueOnce(completed)
    const store = useSprintsStore()
    store.sprints = [activeSprint]
    await store.completeSprint('TEST', activeSprint.id, { next_sprint_id: 'sprint-next' })
    expect(mockComplete).toHaveBeenCalledWith('TEST', activeSprint.id, { next_sprint_id: 'sprint-next' })
    const sprint0 = store.sprints[0]
    expect(sprint0).toBeDefined()
    expect(sprint0!.status).toBe('completed')
  })

  test('deleteSprint removes the sprint from the list', async () => {
    mockDelete.mockResolvedValueOnce(undefined)
    const store = useSprintsStore()
    store.sprints = [sprint]
    await store.deleteSprint('TEST', sprint.id)
    expect(store.sprints).toHaveLength(0)
  })

  test('fetchBurndown stores burndown data and returns it', async () => {
    mockBurndown.mockResolvedValueOnce(burndown)
    const store = useSprintsStore()
    const result = await store.fetchBurndown('TEST', activeSprint.id)
    expect(result.sprint_id).toBe('sprint-active')
    expect(store.burndownData?.total_points).toBe(20)
  })

  test('fetchBurndown sets error on failure', async () => {
    mockBurndown.mockRejectedValueOnce({ message: 'Sprint not active' })
    const store = useSprintsStore()
    await expect(store.fetchBurndown('TEST', sprint.id)).rejects.toBeDefined()
    expect(store.error).toBe('Sprint not active')
  })
})

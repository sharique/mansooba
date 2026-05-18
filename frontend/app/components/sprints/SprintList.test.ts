// @vitest-environment happy-dom
import { shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, it, expect, vi } from 'vitest'
import SprintList from './SprintList.vue'
import type { Sprint } from '~/types/domain.types'

vi.mock('~/services/issues.service', () => ({
  issuesService: { update: vi.fn() },
}))

const mockShowSuccess = vi.fn()
const mockShowError = vi.fn()
const mockSprintsStore = {
  sprints: [] as Sprint[],
  openSprints: [] as Sprint[],
  activeSprint: null as Sprint | null,
  sprintIssues: {} as Record<string, Sprint[]>,
  fetchSprintIssues: vi.fn(),
  startSprint: vi.fn(),
  deleteSprint: vi.fn(),
  removeFromSprintIssues: vi.fn(),
}

vi.stubGlobal('useSprintsStore', () => mockSprintsStore)
vi.stubGlobal('useToast', () => ({ showSuccess: mockShowSuccess, showError: mockShowError }))

const sprint: Sprint = {
  id: 'sprint-1',
  project_id: 'p-1',
  name: 'Sprint 1',
  goal: '',
  status: 'planning',
  start_date: null,
  end_date: null,
  created_at: '2026-05-01T00:00:00Z',
  updated_at: '2026-05-01T00:00:00Z',
}

describe('SprintList', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockSprintsStore.sprints = []
    mockSprintsStore.activeSprint = null
    mockShowSuccess.mockReset()
    mockShowError.mockReset()
  })

  it('shows "No sprints yet." when store has no sprints', () => {
    const w = shallowMount(SprintList, { props: { projectKey: 'P', canManage: false } })
    expect(w.text()).toContain('No sprints yet.')
  })

  it('does not show "No sprints yet." when sprints exist', () => {
    mockSprintsStore.sprints = [sprint]
    const w = shallowMount(SprintList, { props: { projectKey: 'P', canManage: false } })
    expect(w.text()).not.toContain('No sprints yet.')
  })

  it('shows "+ New Sprint" button when canManage is true', () => {
    const w = shallowMount(SprintList, { props: { projectKey: 'P', canManage: true } })
    expect(w.text()).toContain('+ New Sprint')
  })

  it('hides "+ New Sprint" button when canManage is false', () => {
    const w = shallowMount(SprintList, { props: { projectKey: 'P', canManage: false } })
    expect(w.text()).not.toContain('+ New Sprint')
  })

  it('renders "Sprints" heading', () => {
    const w = shallowMount(SprintList, { props: { projectKey: 'P', canManage: false } })
    expect(w.text()).toContain('Sprints')
  })
})

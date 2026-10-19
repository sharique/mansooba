// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { onMounted, reactive, ref, computed } from 'vue'
import OverviewPage from './index.vue'
import SprintSummary from '~/components/overview/SprintSummary.vue'
import OpenTasks from '~/components/overview/OpenTasks.vue'
import RecentActivity from '~/components/overview/RecentActivity.vue'
import { useOverviewStore } from '~/stores/overview.store'
import { overviewService } from '~/services/overview.service'

vi.mock('~/services/overview.service', () => ({
  overviewService: { getSprint: vi.fn(), getTasks: vi.fn(), getActivity: vi.fn() },
}))
const svc = vi.mocked(overviewService)

const projects = reactive({
  current: null as Record<string, unknown> | null,
  fetchOne: vi.fn(),
})
const toast = { showSuccess: vi.fn(), showError: vi.fn() }

vi.stubGlobal('useRoute', () => ({ params: { key: 'scr' } }))
vi.stubGlobal('useProjectsStore', () => projects)
vi.stubGlobal('useOverviewStore', useOverviewStore)
vi.stubGlobal('useToast', () => toast)
vi.stubGlobal('useTimeFormatter', () => ({ formatDate: (s?: string) => s ?? '—', formatDateTime: (s: string) => s }))
vi.stubGlobal('ref', ref)
vi.stubGlobal('computed', computed)
vi.stubGlobal('onMounted', onMounted)

const NuxtLinkStub = { props: ['to'], template: '<a :href="to"><slot /></a>' }
const sprint = { id: 3, name: 'Sprint 3', is_overdue: false, todo: { tasks: 1, points: 1 }, in_progress: { tasks: 0, points: 0 }, in_review: { tasks: 0, points: 0 }, done: { tasks: 0, points: 0 }, total: { tasks: 1, points: 1 } }
const openTask = { issue_id: 1, key: 'SCR-1', title: 'First task', status: 'todo', priority: 'low', assignee: null }

const mountIt = () =>
  mount(OverviewPage, {
    global: {
      stubs: { NuxtLink: NuxtLinkStub, UserAvatar: true },
      components: { OverviewSprintSummary: SprintSummary, OverviewOpenTasks: OpenTasks, OverviewRecentActivity: RecentActivity },
    },
  })

describe('overview page', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.resetAllMocks()
    projects.current = { id: 1, key: 'scr', name: 'Scratch', description: 'About scratch', ownerId: 9 }
    projects.fetchOne.mockResolvedValue(undefined)
    svc.getSprint.mockResolvedValue(sprint as never)
    svc.getTasks.mockResolvedValue({ sprint_active: true, items: [openTask] } as never)
    svc.getActivity.mockResolvedValue({ sprint_active: true, items: [] } as never)
  })

  it('shows the project name and description read-only plus the three sections', async () => {
    const w = mountIt()
    await flushPromises()
    expect(w.text()).toContain('Scratch')
    expect(w.text()).toContain('About scratch')
    expect(w.text()).toContain('Sprint 3')
    expect(w.text()).toContain('First task')
    expect(w.text()).toContain('No recent activity')
    expect(svc.getSprint).toHaveBeenCalledWith('scr')
    expect(svc.getTasks).toHaveBeenCalledWith('scr')
    expect(svc.getActivity).toHaveBeenCalledWith('scr', 'all')
    expect(w.findAll('button').some(b => b.text() === 'Edit')).toBe(false)
  })

  it('no longer shows the member list', async () => {
    const w = mountIt()
    await flushPromises()
    expect(w.find('table').exists()).toBe(false)
    expect(w.text()).not.toMatch(/invite|add member/i)
  })

  it('a failing section leaves the others rendered', async () => {
    svc.getTasks.mockRejectedValue(new Error('tasks down'))
    const w = mountIt()
    await flushPromises()
    expect(w.text()).toContain('tasks down')
    expect(w.text()).toContain('Sprint 3')
    expect(w.text()).toContain('No recent activity')
  })

  it('retry reloads only the failed section', async () => {
    svc.getTasks.mockRejectedValueOnce(new Error('tasks down')).mockResolvedValueOnce({ sprint_active: true, items: [openTask] } as never)
    const w = mountIt()
    await flushPromises()
    svc.getSprint.mockClear()
    svc.getActivity.mockClear()
    await w.find('section[aria-labelledby="overview-tasks-title"] button').trigger('click')
    await flushPromises()
    expect(svc.getTasks).toHaveBeenCalledTimes(2)
    expect(svc.getSprint).not.toHaveBeenCalled()
    expect(svc.getActivity).not.toHaveBeenCalled()
    expect(w.text()).toContain('First task')
  })

  it('switching the activity tab reloads only activity, with that scope', async () => {
    const w = mountIt()
    await flushPromises()
    svc.getSprint.mockClear()
    svc.getTasks.mockClear()
    svc.getActivity.mockClear()
    await w.findAll('[role="tab"]')[1]!.trigger('click')
    await flushPromises()
    expect(svc.getActivity).toHaveBeenCalledTimes(1)
    expect(svc.getActivity).toHaveBeenCalledWith('scr', 'sprint')
    expect(svc.getSprint).not.toHaveBeenCalled()
    expect(svc.getTasks).not.toHaveBeenCalled()
    expect(w.findAll('[role="tab"]')[1]!.attributes('aria-selected')).toBe('true')
  })

  it('for someone who cannot see the project, shows no project data', async () => {
    projects.current = null
    projects.fetchOne.mockRejectedValue({ status: 403 })
    svc.getSprint.mockRejectedValue({ data: { message: 'forbidden' } })
    svc.getTasks.mockRejectedValue({ data: { message: 'forbidden' } })
    svc.getActivity.mockRejectedValue({ data: { message: 'forbidden' } })
    const w = mountIt()
    await flushPromises()
    expect(w.text()).not.toContain('Scratch')
    expect(w.text()).not.toContain('About scratch')
    expect(w.text()).not.toContain('Sprint 3')
    expect(w.text()).not.toContain('First task')
    expect(w.text()).toContain('forbidden')
    expect(toast.showError).toHaveBeenCalled()
  })

  it('never shows another project\'s name left over in the store when this project cannot be loaded', async () => {
    projects.current = { id: 2, key: 'other', name: 'Someone Else Project', description: 'private', ownerId: 3 }
    projects.fetchOne.mockRejectedValue({ status: 403 })
    const w = mountIt()
    await flushPromises()
    expect(w.text()).not.toContain('Someone Else Project')
    expect(w.text()).not.toContain('private')
  })

  it('starts from a clean store so another project never shows stale numbers', async () => {
    const store = useOverviewStore()
    store.sprint.data = { name: 'Old project sprint' } as never
    svc.getSprint.mockImplementation(() => new Promise(() => {})) // never answers
    const w = mountIt()
    await flushPromises()
    expect(w.text()).not.toContain('Old project sprint')
  })
})

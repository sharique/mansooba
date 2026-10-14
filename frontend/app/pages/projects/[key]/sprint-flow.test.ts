// @vitest-environment happy-dom
// FR-016: the sprint-membership flow across the backlog and Sprints pages, over
// one real Pinia instance. Only the HTTP service modules are replaced, with a
// small in-memory fake that behaves like the server (0 clears sprint_id).
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import BacklogPage from './backlog.vue'
import SprintsPage from './sprints.vue'
import BacklogList from '~/components/backlog/BacklogList.vue'
import BacklogIssueRow from '~/components/backlog/BacklogIssueRow.vue'
import SprintList from '~/components/sprints/SprintList.vue'
import SprintCard from '~/components/sprints/SprintCard.vue'
import { useProjectsStore } from '~/stores/projects.store'
import { useSprintsStore } from '~/stores/sprints.store'
import { useIssuesStore } from '~/stores/issues.store'
import { useCanManageProject } from '~/composables/useCanManageProject'

const fake = vi.hoisted(() => ({
  issues: [] as { id: number; key: string; title: string; sprint_id: number | null; priority: string; type: string }[],
}))

vi.mock('~/services/backlog.service', () => ({
  backlogService: { getBacklog: async () => fake.issues.filter(i => i.sprint_id === null).map(i => ({ ...i })) },
}))
vi.mock('~/services/projects.service', () => ({
  projectsService: {
    get: async () => ({ id: 1, key: 'TEST', name: 'T', description: '', ownerId: 1 }),
    listMembers: async () => [{ user_id: 1, name: 'Owner', email: 'o@x.com', role: 'admin' }],
  },
}))
vi.mock('~/services/sprints.service', () => ({
  sprintsService: {
    list: async () => [{ id: '1', project_id: '1', name: 'Sprint 1', goal: '', status: 'planning', start_date: null, end_date: null }],
    listIssues: async (_k: string, id: string) => fake.issues.filter(i => i.sprint_id === Number(id)).map(i => ({ ...i })),
  },
}))
vi.mock('~/services/issues.service', () => ({
  issuesService: {
    update: vi.fn(async (_k: string, id: number, data: { sprint_id?: number }) => {
      const issue = fake.issues.find(i => i.id === id)!
      if (data.sprint_id !== undefined) issue.sprint_id = data.sprint_id === 0 ? null : data.sprint_id
      return { ...issue }
    }),
  },
}))

vi.stubGlobal('useRoute', () => ({ params: { key: 'TEST' } }))
vi.stubGlobal('useAuthStore', () => ({ user: { id: 1 } }))
vi.stubGlobal('useProjectsStore', useProjectsStore)
vi.stubGlobal('useSprintsStore', useSprintsStore)
vi.stubGlobal('useIssuesStore', useIssuesStore)
vi.stubGlobal('useCanManageProject', useCanManageProject)
vi.stubGlobal('useToast', () => ({ showSuccess: vi.fn(), showError: vi.fn() }))
vi.stubGlobal('useTimeFormatter', () => ({ formatDate: (d: string) => d }))
vi.stubGlobal('navigateTo', vi.fn())

const components = {
  BacklogList,
  BacklogIssueRow,
  SprintsSprintList: SprintList,
  SprintsSprintCard: SprintCard,
}

const mountPage = (page: object) => mount(page, { global: { components } })

describe('sprint flow across the backlog and Sprints pages', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    fake.issues = [{ id: 7, key: 'TEST-7', title: 'Seven', sprint_id: null, priority: 'low', type: 'task' }]
  })

  it('add on backlog -> visible in sprint -> remove -> back in backlog', async () => {
    const sprintsStore = useSprintsStore()

    // Prime the per-sprint cache (empty) the way a previous visit would.
    const first = mountPage(SprintsPage)
    await flushPromises()
    await first.find('button.btn-ghost.mt-2').trigger('click')
    await flushPromises()
    expect(sprintsStore.sprintIssues['1']).toEqual([])
    first.unmount()

    // (a)(b) backlog shows issue 7; adding it to the sprint removes it from the list.
    const backlog = mountPage(BacklogPage)
    await flushPromises()
    expect(backlog.text()).toContain('Seven')
    await backlog.find('select[title="Add to sprint"]').setValue('1')
    await flushPromises()
    expect(backlog.text()).not.toContain('Seven')
    expect(sprintsStore.sprintIssues['1']).toBeUndefined() // stale cache dropped
    backlog.unmount()

    // (c) Sprints page lists it after expanding.
    const sprints = mountPage(SprintsPage)
    await flushPromises()
    await sprints.find('button.btn-ghost.mt-2').trigger('click')
    await flushPromises()
    expect(sprints.text()).toContain('Seven')

    // (d) manager removes it -> server sees sprint_id 0.
    await sprints.find('button[title="Remove from sprint"]').trigger('click')
    await flushPromises()
    const { issuesService } = await import('~/services/issues.service')
    expect(issuesService.update).toHaveBeenLastCalledWith('TEST', 7, { sprint_id: 0 })
    sprints.unmount()

    // (e) back in the backlog.
    const again = mountPage(BacklogPage)
    await flushPromises()
    expect(again.text()).toContain('Seven')
  })
})

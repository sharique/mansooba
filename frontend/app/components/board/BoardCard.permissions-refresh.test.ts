// @vitest-environment happy-dom
// Real store and composable: after a refused change, refreshing the project makes
// the controls follow the new role without a reload.
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import BoardCard from './BoardCard.vue'
import { useProjectsStore } from '~/stores/projects.store'
import { useProjectPermissions } from '~/composables/useProjectPermissions'

const get = vi.hoisted(() => vi.fn())
vi.mock('~/services/projects.service', () => ({ projectsService: { get } }))
vi.stubGlobal('useProjectsStore', useProjectsStore)
vi.stubGlobal('useProjectPermissions', useProjectPermissions)
vi.stubGlobal('navigateTo', vi.fn())

const project = (myRole: string, permissions: string[]) =>
  ({ id: 1, key: 'proj', name: 'P', description: '', ownerId: 1, myRole, permissions })

describe('stale permissions', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    get.mockReset()
  })

  it('the move control is switched off once a refresh shows the person became a viewer', async () => {
    const store = useProjectsStore()
    store.current = project('member', ['project.view', 'task.write']) as never
    const w = mount(BoardCard, {
      props: {
        issue: { id: 3, key: 'PROJ-3', project_id: 1, title: 'Fix', description: '', type: 'task', status: 'todo', priority: 'high', reporter_id: 1, attachment_count: 0 },
        projectKey: 'proj',
      },
      global: { stubs: { UserAvatar: true } },
    })
    const toggle = () => w.find('button[aria-label^="Move"]')
    expect(toggle().attributes('aria-disabled')).toBeUndefined()

    get.mockResolvedValueOnce(project('viewer', ['project.view']))
    await store.refreshPermissions('proj')
    await flushPromises()

    expect(toggle().attributes('aria-disabled')).toBe('true')
    expect(w.classes()).toContain('no-drag')
  })
})

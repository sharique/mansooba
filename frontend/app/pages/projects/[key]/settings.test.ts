// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import SettingsPage from './settings.vue'
import { stubPermissions, ADMIN, MEMBER, VIEWER } from '../../../../test-helpers/permissions'

const perms = stubPermissions(ADMIN, 'admin')
const projects = reactive({
  current: { id: 1, key: 'scr', name: 'Scratch', description: '' } as Record<string, unknown> | null,
  fetchOne: vi.fn(async () => {}),
})
vi.stubGlobal('useRoute', () => ({ params: { key: 'scr' } }))
vi.stubGlobal('useProjectsStore', () => projects)
vi.stubGlobal('onMounted', (fn: () => void) => fn())

const mountIt = () => mount(SettingsPage, {
  global: {
    stubs: { LabelsLabelManager: true, NuxtLink: true },
    components: { ProjectsDeleteProjectDialog: { props: ['projectKey', 'projectName'], template: '<div data-delete-dialog>{{ projectName }}</div>' } },
  },
})

describe('settings page', () => {
  beforeEach(() => {
    perms.permissions = ADMIN
    perms.myRole = 'admin'
    projects.current = { id: 1, key: 'scr', name: 'Scratch', description: '' }
    projects.fetchOne.mockClear()
  })

  it('loads the project so the permissions are known', async () => {
    mountIt()
    await flushPromises()
    expect(projects.fetchOne).toHaveBeenCalledWith('scr')
  })

  it('offers project deletion to someone with project.delete', async () => {
    const w = mountIt()
    await flushPromises()
    expect(w.find('[data-delete-dialog]').text()).toBe('Scratch')
  })

  it.each([['member', MEMBER], ['viewer', VIEWER]])('does not offer project deletion to a %s', async (role, perm) => {
    perms.permissions = perm
    perms.myRole = role
    const w = mountIt()
    await flushPromises()
    expect(w.find('[data-delete-dialog]').exists()).toBe(false)
  })
})

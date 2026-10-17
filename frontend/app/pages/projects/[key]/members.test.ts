// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import MembersPage from './members.vue'

const projects = reactive({
  current: { id: 1, key: 'scr', name: 'Scratch', description: 'A scratch project', ownerId: 9 } as Record<string, unknown> | null,
  fetchOne: vi.fn(async () => {}),
})
const toast = { showSuccess: vi.fn(), showError: vi.fn() }

vi.stubGlobal('useRoute', () => ({ params: { key: 'scr' } }))
vi.stubGlobal('useProjectsStore', () => projects)
vi.stubGlobal('useToast', () => toast)
vi.stubGlobal('ref', (await import('vue')).ref)
vi.stubGlobal('onMounted', (await import('vue')).onMounted)

const MemberListStub = { props: ['projectKey', 'ownerId'], template: '<div data-member-list>{{ projectKey }}/{{ ownerId }}</div>' }
const FormStub = { props: ['project'], emits: ['saved', 'cancel'], template: '<div data-project-form />' }
const NuxtLinkStub = { props: ['to'], template: '<a :href="to"><slot /></a>' }

const mountIt = () =>
  mount(MembersPage, {
    global: {
      stubs: { NuxtLink: NuxtLinkStub },
      components: { ProjectsMemberList: MemberListStub, ProjectsProjectForm: FormStub },
    },
  })

describe('members page', () => {
  beforeEach(() => {
    projects.current = { id: 1, key: 'scr', name: 'Scratch', description: 'A scratch project', ownerId: 9 }
    projects.fetchOne.mockClear()
    toast.showSuccess.mockClear()
    toast.showError.mockClear()
  })

  it('shows the breadcrumb, name, description and the member list for the project', async () => {
    const w = mountIt()
    await flushPromises()
    expect(projects.fetchOne).toHaveBeenCalledWith('scr')
    expect(w.text()).toContain('Scratch')
    expect(w.text()).toContain('A scratch project')
    const crumbs = w.find('.breadcrumbs').text()
    expect(crumbs).toContain('Projects')
    expect(crumbs).toContain('scr')
    expect(crumbs).toContain('Members')
    expect(w.find('a[href="/projects/scr"]').exists()).toBe(true)
    expect(w.find('[data-member-list]').text()).toBe('scr/9')
  })

  it('keeps the Edit button and the edit dialog with the project form', async () => {
    const w = mountIt()
    await flushPromises()
    const edit = w.findAll('button').find(b => b.text() === 'Edit')
    expect(edit).toBeTruthy()
    expect(w.find('dialog [data-project-form]').exists()).toBe(true)
  })

  it('saving the project form closes the dialog and confirms', async () => {
    const w = mountIt()
    await flushPromises()
    w.findComponent(FormStub).vm.$emit('saved', { name: 'Renamed' })
    await flushPromises()
    expect(toast.showSuccess).toHaveBeenCalledWith('Project "Renamed" updated')
  })

  it('reports a failed load', async () => {
    projects.fetchOne.mockRejectedValueOnce(new Error('403'))
    mountIt()
    await flushPromises()
    expect(toast.showError).toHaveBeenCalledWith('Failed to load project')
  })

  it('does not render any overview section', async () => {
    const w = mountIt()
    await flushPromises()
    expect(w.text()).not.toMatch(/current sprint|open tasks|recent activity/i)
  })
})

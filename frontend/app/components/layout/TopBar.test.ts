// @vitest-environment happy-dom
import { shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, it, expect, vi } from 'vitest'
import TopBar from './TopBar.vue'

// TopBar explicitly imports useAuthStore and useProjectsStore, so vi.mock is needed
const mockAuthStore = {
  profile: null as { name: string; is_admin: boolean } | null,
  user: null,
  isAdmin: false,
  clearAuth: vi.fn(),
}

const mockProjectsStore = {
  projects: [] as { id: number; key: string; name: string }[],
}

vi.mock('~/stores/auth.store', () => ({
  useAuthStore: () => mockAuthStore,
}))

vi.mock('~/stores/projects.store', () => ({
  useProjectsStore: () => mockProjectsStore,
}))

vi.stubGlobal('useRouter', () => ({ push: vi.fn() }))
vi.stubGlobal('useToast', () => ({ showSuccess: vi.fn(), showError: vi.fn() }))
vi.stubGlobal('inject', () => vi.fn())

const globalStubs = {
  UserAvatar: true,
  LayoutNotificationBell: true,
  LayoutThemeToggle: true,
  IssuesCreateIssueModal: true,
  Icon: true,
  NuxtLink: { template: '<a><slot /></a>' },
}

describe('TopBar create control role-aware rendering', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockAuthStore.profile = null
    mockAuthStore.isAdmin = false
    mockProjectsStore.projects = []
  })

  it('Admin with projects sees a dropdown with two options', () => {
    mockAuthStore.profile = { name: 'Admin', is_admin: true }
    mockAuthStore.isAdmin = true
    mockProjectsStore.projects = [{ id: 1, key: 'PROJ', name: 'My Project' }]

    const w = shallowMount(TopBar, { global: { stubs: globalStubs } })
    expect(w.find('[data-testid="create-dropdown"]').exists()).toBe(true)
    expect(w.find('[data-testid="create-task-btn"]').exists()).toBe(false)
  })

  it('Non-admin member with projects sees a single create task button', () => {
    mockAuthStore.profile = { name: 'Alice', is_admin: false }
    mockAuthStore.isAdmin = false
    mockProjectsStore.projects = [{ id: 1, key: 'PROJ', name: 'My Project' }]

    const w = shallowMount(TopBar, { global: { stubs: globalStubs } })
    expect(w.find('[data-testid="create-dropdown"]').exists()).toBe(false)
    expect(w.find('[data-testid="create-task-btn"]').exists()).toBe(true)
  })

  it('User with no projects and no admin sees no create control', () => {
    mockAuthStore.profile = { name: 'Bob', is_admin: false }
    mockAuthStore.isAdmin = false
    mockProjectsStore.projects = []

    const w = shallowMount(TopBar, { global: { stubs: globalStubs } })
    expect(w.find('[data-testid="create-dropdown"]').exists()).toBe(false)
    expect(w.find('[data-testid="create-task-btn"]').exists()).toBe(false)
  })
})

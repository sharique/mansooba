// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ProjectsPage from './projects.vue'
import { useAdminProjectsStore } from '~/stores/admin-projects.store'

const navigate = vi.fn()
const toast = { showSuccess: vi.fn(), showError: vi.fn() }
const auth = { isAdmin: true }
const list = vi.fn()
const claim = vi.fn()

vi.stubGlobal('navigateTo', navigate)
vi.stubGlobal('useToast', () => toast)
vi.stubGlobal('useAuthStore', () => auth)
vi.stubGlobal('useAdminProjectsStore', useAdminProjectsStore)
vi.stubGlobal('onMounted', (fn: () => void) => fn())
vi.mock('~/services/admin.service', () => ({
  adminService: { listProjectsWithoutAdmin: () => list(), claimProjectAdmin: (k: string) => claim(k) },
}))

const rows = [{ id: 1, key: 'old', name: 'Old Project', member_count: 3 }]
const mountIt = () => mount(ProjectsPage, { global: { stubs: { Icon: true } } })

describe('system/projects page', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    auth.isAdmin = true
    list.mockReset().mockResolvedValue(rows)
    claim.mockReset().mockResolvedValue({})
    navigate.mockReset()
    toast.showSuccess.mockReset()
    toast.showError.mockReset()
    vi.stubGlobal('confirm', vi.fn(() => true))
  })

  it('sends someone who is not a system administrator away and loads nothing', async () => {
    auth.isAdmin = false
    mountIt()
    await flushPromises()
    expect(navigate).toHaveBeenCalledWith('/')
    expect(list).not.toHaveBeenCalled()
  })

  it('lists key, name and member count for each project', async () => {
    const w = mountIt()
    await flushPromises()
    const cells = w.findAll('tbody tr td').map(c => c.text())
    expect(cells).toEqual(expect.arrayContaining(['old', 'Old Project', '3']))
  })

  it('says every project has an admin when the list is empty', async () => {
    list.mockResolvedValue([])
    const w = mountIt()
    await flushPromises()
    expect(w.text()).toContain('Every project has an active admin.')
    expect(w.find('table').exists()).toBe(false)
  })

  it('asks first, then takes admin and drops the row', async () => {
    const w = mountIt()
    await flushPromises()
    await w.find('button[aria-label="Take admin of Old Project"]').trigger('click')
    await flushPromises()
    expect(globalThis.confirm).toHaveBeenCalled()
    expect(claim).toHaveBeenCalledWith('old')
    expect(toast.showSuccess).toHaveBeenCalled()
    expect(w.text()).toContain('Every project has an active admin.')
  })

  it('does nothing when the confirmation is declined', async () => {
    vi.stubGlobal('confirm', vi.fn(() => false))
    const w = mountIt()
    await flushPromises()
    await w.find('button[aria-label="Take admin of Old Project"]').trigger('click')
    expect(claim).not.toHaveBeenCalled()
  })

  it('shows the server reason when someone else became admin meanwhile (409)', async () => {
    claim.mockRejectedValueOnce({ status: 409, data: { message: 'This project already has an active admin' } })
    const w = mountIt()
    await flushPromises()
    await w.find('button[aria-label="Take admin of Old Project"]').trigger('click')
    await flushPromises()
    expect(toast.showError).toHaveBeenCalledWith('This project already has an active admin')
  })
})

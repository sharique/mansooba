// @vitest-environment happy-dom
import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { useBoardStore } from '~/stores/board.store'
import { beforeEach, describe, it, expect, vi } from 'vitest'
import BoardPage from './board.vue'
import BoardColumn from '~/components/board/BoardColumn.vue'
import { issuesService } from '~/services/issues.service'
import { boardService } from '~/services/board.service'

vi.mock('~/services/board.service', () => ({
  boardService: { getBoard: vi.fn().mockResolvedValue({ columns: [] }) },
}))
vi.mock('~/services/issues.service', () => ({
  issuesService: { update: vi.fn().mockResolvedValue({}) },
}))

vi.stubGlobal('useRoute', () => ({ params: { key: 'TEST' } }))
vi.stubGlobal('useIssuesStore', () => ({ update: vi.fn() }))
vi.stubGlobal('useToast', () => ({ showSuccess: vi.fn(), showError: vi.fn() }))

describe('board page', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('does not render a burndown chart (it lives on the Reports page)', async () => {
    const w = shallowMount(BoardPage)
    await flushPromises()
    expect(w.find('section.mb-6').exists()).toBe(false)
    expect(w.find('chartsburndownchart').exists()).toBe(false)
  })

  it('shows "Backlog →" navigation link', async () => {
    const w = shallowMount(BoardPage)
    await flushPromises()
    expect(w.text()).toContain('Backlog →')
  })

  it('links to the Sprints page next to the Backlog link', async () => {
    const w = shallowMount(BoardPage)
    await flushPromises()
    expect(w.find('nuxtlink[to="/projects/TEST/sprints"]').exists()).toBe(true)
    expect(w.text()).toContain('Sprints →')
  })

  describe('board store wiring', () => {
    const issue = { id: 5, key: 'TEST-5', project_id: 1, title: 'Task', status: 'todo', priority: 'high', attachment_count: 0 }
    // Nuxt auto-imports the column; outside Nuxt it has to be registered for the page to find it.
    const mountPage = () => shallowMount(BoardPage, { global: { components: { BoardColumn } } })
    const withBoard = () => vi.mocked(boardService.getBoard).mockResolvedValue({
      columns: ['todo', 'in_progress', 'in_review', 'done'].map(status => ({ status, issues: status === 'todo' ? [issue] : [] })),
    } as never)

    it('renders one column per board column, loaded through the store', async () => {
      withBoard()
      const w = mountPage()
      await flushPromises()
      expect(w.findAllComponents(BoardColumn)).toHaveLength(4)
      expect(boardService.getBoard).toHaveBeenCalledWith('TEST')
    })

    it('saves a move when a column reports a dropped card', async () => {
      withBoard()
      const w = mountPage()
      await flushPromises()
      w.findAllComponents(BoardColumn)[1]!.vm.$emit('moved', { issueId: 5, toStatus: 'in_progress' })
      await flushPromises()
      expect(issuesService.update).toHaveBeenCalledWith('TEST', 5, { status: 'in_progress' })
    })
  })

  describe('refresh when the person comes back', () => {
    const setVisibility = (state: 'visible' | 'hidden') =>
      Object.defineProperty(document, 'visibilityState', { value: state, configurable: true })

    const mountWithSpy = async () => {
      const store = useBoardStore()
      const refresh = vi.spyOn(store, 'refreshIfStale').mockResolvedValue()
      const w = shallowMount(BoardPage)
      await flushPromises()
      return { w, refresh }
    }

    it('refreshes when the tab becomes visible again', async () => {
      const { refresh } = await mountWithSpy()
      setVisibility('visible')
      document.dispatchEvent(new Event('visibilitychange'))
      expect(refresh).toHaveBeenCalledWith('TEST')
    })

    it('does nothing when the tab is hidden', async () => {
      const { refresh } = await mountWithSpy()
      setVisibility('hidden')
      document.dispatchEvent(new Event('visibilitychange'))
      expect(refresh).not.toHaveBeenCalled()
      setVisibility('visible')
    })

    it('refreshes when the window gets focus back', async () => {
      const { refresh } = await mountWithSpy()
      window.dispatchEvent(new Event('focus'))
      expect(refresh).toHaveBeenCalledWith('TEST')
    })

    it('stops listening when the page goes away', async () => {
      const { w, refresh } = await mountWithSpy()
      w.unmount()
      setVisibility('visible')
      document.dispatchEvent(new Event('visibilitychange'))
      window.dispatchEvent(new Event('focus'))
      expect(refresh).not.toHaveBeenCalled()
    })
  })

  describe('announcements for screen readers', () => {
    it('has a polite live region that reads out the store\'s announcement', async () => {
      const store = useBoardStore()
      const w = shallowMount(BoardPage)
      await flushPromises()
      const region = w.find('[role="status"]')
      expect(region.exists()).toBe(true)
      expect(region.attributes('aria-live')).toBe('polite')
      expect(region.classes()).toContain('sr-only')
      expect(region.text()).toBe('')

      store.announcement = 'TEST-5 moved to Done'
      await flushPromises()
      expect(region.text()).toBe('TEST-5 moved to Done')
    })
  })

  it('calls the page "Sprint board" in its title and breadcrumb', async () => {
    const w = shallowMount(BoardPage)
    await flushPromises()
    expect(w.find('h1').text()).toBe('Sprint board')
    expect(w.find('.breadcrumbs li:last-child').text()).toBe('Sprint board')
  })
})

// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import IssueForm from './IssueForm.vue'

const mockCreate = vi.fn()

vi.mock('~/stores/issues.store', () => ({
  useIssuesStore: () => ({ create: mockCreate, update: vi.fn() }),
}))

vi.mock('~/services/projects.service', () => ({
  projectsService: { listMembers: vi.fn().mockResolvedValue([]) },
}))

vi.stubGlobal('useIssuesStore', () => ({ create: mockCreate, update: vi.fn() }))
vi.stubGlobal('useToast', () => ({ showSuccess: vi.fn(), showError: vi.fn() }))
vi.stubGlobal('onMounted', (fn: () => void) => fn())

describe('IssueForm', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockCreate.mockReset()
  })

  test('does not emit saved when title is empty', async () => {
    const wrapper = mount(IssueForm, {
      props: { projectKey: 'PROJ' },
    })
    await wrapper.find('[data-testid="submit"]').trigger('click')
    expect(wrapper.emitted('saved')).toBeUndefined()
    expect(mockCreate).not.toHaveBeenCalled()
  })
})

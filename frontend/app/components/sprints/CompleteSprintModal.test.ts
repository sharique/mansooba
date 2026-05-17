// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import CompleteSprintModal from './CompleteSprintModal.vue'
import type { Sprint } from '~/types/domain.types'

const mockCompleteSprint = vi.fn()
const mockShowSuccess = vi.fn()
const mockShowError = vi.fn()

vi.stubGlobal('useSprintsStore', () => ({ completeSprint: mockCompleteSprint }))
vi.stubGlobal('useToast', () => ({ showSuccess: mockShowSuccess, showError: mockShowError }))

const activeSprint: Sprint = {
  id: 'sprint-active',
  project_id: 'p-1',
  name: 'Sprint 1',
  goal: '',
  status: 'active',
  start_date: '2026-05-01',
  end_date: '2026-05-14',
  created_at: '2026-05-01T00:00:00Z',
  updated_at: '2026-05-01T00:00:00Z',
}

describe('CompleteSprintModal', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockCompleteSprint.mockReset()
    mockShowSuccess.mockReset()
    mockShowError.mockReset()
  })

  test('emits cancel when backdrop is clicked', async () => {
    const wrapper = mount(CompleteSprintModal, {
      props: { projectKey: 'TEST', sprint: activeSprint, otherSprints: [] },
    })
    await wrapper.find('.modal-backdrop').trigger('click')
    expect(wrapper.emitted('cancel')).toBeTruthy()
  })

  test('calls completeSprint with empty payload when backlog option selected', async () => {
    mockCompleteSprint.mockResolvedValue({ ...activeSprint, status: 'completed' })
    const wrapper = mount(CompleteSprintModal, {
      props: { projectKey: 'TEST', sprint: activeSprint, otherSprints: [] },
    })
    await wrapper.find('button.btn-warning').trigger('click')
    expect(mockCompleteSprint).toHaveBeenCalledWith('TEST', 'sprint-active', {})
  })

  test('calls completeSprint with next_sprint_id when a sprint is selected', async () => {
    const planningSprint: Sprint = {
      ...activeSprint,
      id: 'sprint-next',
      name: 'Sprint 2',
      status: 'planning',
    }
    mockCompleteSprint.mockResolvedValue({ ...activeSprint, status: 'completed' })
    const wrapper = mount(CompleteSprintModal, {
      props: { projectKey: 'TEST', sprint: activeSprint, otherSprints: [planningSprint] },
    })
    await wrapper.find('select').setValue('sprint-next')
    await wrapper.find('button.btn-warning').trigger('click')
    expect(mockCompleteSprint).toHaveBeenCalledWith('TEST', 'sprint-active', { next_sprint_id: 'sprint-next' })
  })

  test('emits completed and shows success toast on success', async () => {
    const completed: Sprint = { ...activeSprint, status: 'completed' }
    mockCompleteSprint.mockResolvedValue(completed)
    const wrapper = mount(CompleteSprintModal, {
      props: { projectKey: 'TEST', sprint: activeSprint, otherSprints: [] },
    })
    await wrapper.find('button.btn-warning').trigger('click')
    await flushPromises()
    expect(wrapper.emitted('completed')).toBeTruthy()
    expect(mockShowSuccess).toHaveBeenCalledWith('"Sprint 1" completed')
  })

  test('calls showError and does not emit completed when completeSprint rejects', async () => {
    mockCompleteSprint.mockRejectedValue({ data: { message: 'Sprint already completed' } })
    const wrapper = mount(CompleteSprintModal, {
      props: { projectKey: 'TEST', sprint: activeSprint, otherSprints: [] },
    })
    await wrapper.find('button.btn-warning').trigger('click')
    await flushPromises()
    expect(mockShowError).toHaveBeenCalledWith('Sprint already completed')
    expect(wrapper.emitted('completed')).toBeFalsy()
  })
})

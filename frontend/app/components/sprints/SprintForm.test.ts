// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, it, expect, vi } from 'vitest'
import SprintForm from './SprintForm.vue'
import type { Sprint } from '~/types/domain.types'

const mockCreateSprint = vi.fn()
const mockUpdateSprint = vi.fn()
const mockShowSuccess = vi.fn()
const mockShowError = vi.fn()

vi.stubGlobal('useSprintsStore', () => ({
  createSprint: mockCreateSprint,
  updateSprint: mockUpdateSprint,
}))
vi.stubGlobal('useToast', () => ({ showSuccess: mockShowSuccess, showError: mockShowError }))

const editSprint: Sprint = {
  id: 'sprint-1',
  project_id: 'p-1',
  name: 'Sprint 1',
  goal: 'Some goal',
  status: 'planning',
  start_date: '2026-05-01T00:00:00Z',
  end_date: '2026-05-14T00:00:00Z',
  created_at: '2026-05-01T00:00:00Z',
  updated_at: '2026-05-01T00:00:00Z',
}

describe('SprintForm', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockCreateSprint.mockReset()
    mockUpdateSprint.mockReset()
    mockShowSuccess.mockReset()
    mockShowError.mockReset()
  })

  it('shows "Create Sprint" title and "Create" button in create mode', () => {
    const w = mount(SprintForm, { props: { projectKey: 'P' } })
    expect(w.text()).toContain('Create Sprint')
    expect(w.find('button[type="submit"]').text()).toContain('Create')
  })

  it('shows "Edit Sprint" title and "Save" button in edit mode', () => {
    const w = mount(SprintForm, { props: { projectKey: 'P', sprint: editSprint } })
    expect(w.text()).toContain('Edit Sprint')
    expect(w.find('button[type="submit"]').text()).toContain('Save')
  })

  it('prefills name field with sprint data in edit mode', () => {
    const w = mount(SprintForm, { props: { projectKey: 'P', sprint: editSprint } })
    const input = w.find('input[type="text"]').element as HTMLInputElement
    expect(input.value).toBe('Sprint 1')
  })

  it('emits cancel when Cancel button is clicked', async () => {
    const w = mount(SprintForm, { props: { projectKey: 'P' } })
    await w.find('button[type="button"]').trigger('click')
    expect(w.emitted('cancel')).toBeTruthy()
  })

  it('calls createSprint and emits saved on valid create submit', async () => {
    const created: Sprint = { ...editSprint, id: 'new-sprint' }
    mockCreateSprint.mockResolvedValue(created)
    const w = mount(SprintForm, { props: { projectKey: 'P' } })
    await w.find('input[type="text"]').setValue('New Sprint')
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(mockCreateSprint).toHaveBeenCalledWith('P', expect.objectContaining({ name: 'New Sprint' }))
    expect(w.emitted('saved')).toBeTruthy()
  })

  it('calls updateSprint and emits saved in edit mode', async () => {
    mockUpdateSprint.mockResolvedValue({ ...editSprint, name: 'Updated Sprint' })
    const w = mount(SprintForm, { props: { projectKey: 'P', sprint: editSprint } })
    await w.find('input[type="text"]').setValue('Updated Sprint')
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(mockUpdateSprint).toHaveBeenCalledWith('P', 'sprint-1', expect.any(Object))
    expect(w.emitted('saved')).toBeTruthy()
  })

  it('calls showError and does not emit saved when createSprint rejects', async () => {
    mockCreateSprint.mockRejectedValue({ data: { message: 'Sprint name taken' } })
    const w = mount(SprintForm, { props: { projectKey: 'P' } })
    await w.find('input[type="text"]').setValue('Conflict Sprint')
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(mockShowError).toHaveBeenCalledWith('Sprint name taken')
    expect(w.emitted('saved')).toBeFalsy()
  })
})

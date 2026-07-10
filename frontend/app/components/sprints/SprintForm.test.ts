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

// Dates computed relative to "today" so this fixture never goes stale and
// fail the form's own "no past dates" validation as the calendar moves on.
function daysFromToday(offset: number): string {
  const d = new Date()
  d.setDate(d.getDate() + offset)
  return d.toISOString().slice(0, 10)
}

const futureStart = daysFromToday(7)
const futureEnd = daysFromToday(14)

const editSprint: Sprint = {
  id: 'sprint-1',
  project_id: 'p-1',
  name: 'Sprint 1',
  goal: 'Some goal',
  status: 'planning',
  start_date: `${futureStart}T00:00:00Z`,
  end_date: `${futureEnd}T00:00:00Z`,
  created_at: '2026-05-01T00:00:00Z',
  updated_at: '2026-05-01T00:00:00Z',
}

async function fillValidDates(w: ReturnType<typeof mount>) {
  const dateInputs = w.findAll('input[type="date"]')
  await dateInputs[0]!.setValue(futureStart)
  await dateInputs[1]!.setValue(futureEnd)
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

  it('prefills name and date fields with sprint data in edit mode', () => {
    const w = mount(SprintForm, { props: { projectKey: 'P', sprint: editSprint } })
    const nameInput = w.find('input[type="text"]').element as HTMLInputElement
    expect(nameInput.value).toBe('Sprint 1')
    const dateInputs = w.findAll('input[type="date"]')
    expect((dateInputs[0]!.element as HTMLInputElement).value).toBe(futureStart)
    expect((dateInputs[1]!.element as HTMLInputElement).value).toBe(futureEnd)
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
    await fillValidDates(w)
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
    await fillValidDates(w)
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(mockShowError).toHaveBeenCalledWith('Sprint name taken')
    expect(w.emitted('saved')).toBeFalsy()
  })

  // ── Validation ────────────────────────────────────────────────────────────

  it('blocks submit and shows an error when name is blank', async () => {
    const w = mount(SprintForm, { props: { projectKey: 'P' } })
    await fillValidDates(w)
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(mockCreateSprint).not.toHaveBeenCalled()
    expect(w.text()).toContain('Sprint name is required')
  })

  it('blocks submit and shows an error when start date is blank', async () => {
    const w = mount(SprintForm, { props: { projectKey: 'P' } })
    await w.find('input[type="text"]').setValue('No Start Date')
    await w.findAll('input[type="date"]')[1]!.setValue(futureEnd)
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(mockCreateSprint).not.toHaveBeenCalled()
    expect(w.text()).toContain('Start date is required')
  })

  it('blocks submit and shows an error when end date is blank', async () => {
    const w = mount(SprintForm, { props: { projectKey: 'P' } })
    await w.find('input[type="text"]').setValue('No End Date')
    await w.findAll('input[type="date"]')[0]!.setValue(futureStart)
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(mockCreateSprint).not.toHaveBeenCalled()
    expect(w.text()).toContain('End date is required')
  })

  it('blocks submit and shows an error when start date is in the past', async () => {
    const w = mount(SprintForm, { props: { projectKey: 'P' } })
    await w.find('input[type="text"]').setValue('Past Start')
    const dateInputs = w.findAll('input[type="date"]')
    await dateInputs[0]!.setValue(daysFromToday(-5))
    await dateInputs[1]!.setValue(futureEnd)
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(mockCreateSprint).not.toHaveBeenCalled()
    expect(w.text()).toContain("Start date can't be in the past")
  })

  it('blocks submit and shows an error when end date is before start date', async () => {
    const w = mount(SprintForm, { props: { projectKey: 'P' } })
    await w.find('input[type="text"]').setValue('Backwards Dates')
    const dateInputs = w.findAll('input[type="date"]')
    await dateInputs[0]!.setValue(futureEnd)
    await dateInputs[1]!.setValue(futureStart)
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(mockCreateSprint).not.toHaveBeenCalled()
    expect(w.text()).toContain('End date must be after start date')
  })

  it('blocks submit when end date equals start date', async () => {
    const w = mount(SprintForm, { props: { projectKey: 'P' } })
    await w.find('input[type="text"]').setValue('Same Day Sprint')
    const dateInputs = w.findAll('input[type="date"]')
    await dateInputs[0]!.setValue(futureStart)
    await dateInputs[1]!.setValue(futureStart)
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(mockCreateSprint).not.toHaveBeenCalled()
    expect(w.text()).toContain('End date must be after start date')
  })
})

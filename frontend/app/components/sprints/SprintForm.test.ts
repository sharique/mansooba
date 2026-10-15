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

const otherSprints: Sprint[] = []
vi.stubGlobal('useSprintsStore', () => ({
  sprints: otherSprints,
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

  describe('editing a sprint that already started', () => {
    const startedSprint: Sprint = {
      ...editSprint,
      status: 'active',
      start_date: `${daysFromToday(-10)}T00:00:00Z`,
      end_date: `${daysFromToday(4)}T00:00:00Z`,
    }

    it('lets the unchanged past start date stay and saves only the new end date', async () => {
      mockUpdateSprint.mockResolvedValueOnce(startedSprint)
      const w = mount(SprintForm, { props: { projectKey: 'P', sprint: startedSprint } })
      const dateInputs = w.findAll('input[type="date"]')
      expect(dateInputs[0]!.attributes('min')).toBeUndefined()
      await dateInputs[1]!.setValue(daysFromToday(17)) // 27 days after the start
      await w.find('form').trigger('submit')
      await flushPromises()
      expect(w.text()).not.toContain("can't be in the past")
      expect(mockUpdateSprint).toHaveBeenCalledWith('P', 'sprint-1', { end_date: `${daysFromToday(17)}T00:00:00Z` })
    })

    it('still rejects a start date the user changes to the past, with a clear message', async () => {
      // A planning sprint whose start date has slipped into the past: it can still be edited.
      const w = mount(SprintForm, { props: { projectKey: 'P', sprint: { ...startedSprint, status: 'planning' } } })
      await w.findAll('input[type="date"]')[0]!.setValue(daysFromToday(-3))
      await w.find('form').trigger('submit')
      await flushPromises()
      expect(mockUpdateSprint).not.toHaveBeenCalled()
      expect(w.text()).toContain("Start date can't be in the past")
    })

    it('uses its own messages, not the browser\'s native date validation', () => {
      const w = mount(SprintForm, { props: { projectKey: 'P', sprint: startedSprint } })
      expect(w.find('form').attributes('novalidate')).toBeDefined()
    })
  })

  describe('overlap warning (never blocks saving)', () => {
    const other = (over: Partial<Sprint>): Sprint => ({
      ...editSprint, id: 'other', name: 'Sprint Other', status: 'planning',
      start_date: `${daysFromToday(10)}T00:00:00Z`, end_date: `${daysFromToday(20)}T00:00:00Z`, ...over,
    })
    const fillAndSubmit = async (w: ReturnType<typeof mount>, start: string, end: string) => {
      await w.find('input[type="text"]').setValue('New')
      const d = w.findAll('input[type="date"]')
      await d[0]!.setValue(start)
      await d[1]!.setValue(end)
      await w.find('form').trigger('submit')
      await flushPromises()
    }
    beforeEach(() => { otherSprints.length = 0 })

    it('warns, names the other sprint, and still saves', async () => {
      otherSprints.push(other({}))
      mockCreateSprint.mockResolvedValueOnce(editSprint)
      const w = mount(SprintForm, { props: { projectKey: 'P' } })
      await fillAndSubmit(w, daysFromToday(15), daysFromToday(25))
      expect(w.find('.alert-warning').text()).toContain('Sprint Other')
      expect(w.find('.alert-warning').text()).toContain('overlap')
      expect(mockCreateSprint).toHaveBeenCalled()
    })

    it('shows the warning as soon as the dates are chosen, before submit', async () => {
      otherSprints.push(other({}))
      const w = mount(SprintForm, { props: { projectKey: 'P' } })
      const d = w.findAll('input[type="date"]')
      await d[0]!.setValue(daysFromToday(15))
      await d[1]!.setValue(daysFromToday(25))
      expect(w.find('.alert-warning').exists()).toBe(true)
    })

    it('does not warn for non-overlapping dates, completed sprints, itself, or sprints without dates', async () => {
      otherSprints.push(
        other({ start_date: `${daysFromToday(30)}T00:00:00Z`, end_date: `${daysFromToday(35)}T00:00:00Z` }),
        other({ id: 'done', status: 'completed' }),
        other({ id: 'sprint-1' }), // the sprint being edited
        other({ id: 'nodates', start_date: null, end_date: null }),
      )
      const w = mount(SprintForm, { props: { projectKey: 'P', sprint: editSprint } })
      expect(w.find('.alert-warning').exists()).toBe(false)
    })

    it('treats sharing only a boundary day as back-to-back, not an overlap', async () => {
      otherSprints.push(other({ start_date: `${daysFromToday(20)}T00:00:00Z`, end_date: `${daysFromToday(25)}T00:00:00Z` }))
      const w = mount(SprintForm, { props: { projectKey: 'P' } })
      const d = w.findAll('input[type="date"]')
      await d[0]!.setValue(daysFromToday(10))
      await d[1]!.setValue(daysFromToday(20))
      expect(w.find('.alert-warning').exists()).toBe(false)
    })

    it('warns when editing an active sprint whose end date has passed', () => {
      const w = mount(SprintForm, { props: { projectKey: 'P', sprint: {
        ...editSprint, status: 'active', start_date: `${daysFromToday(-20)}T00:00:00Z`, end_date: `${daysFromToday(-2)}T00:00:00Z`,
      } } })
      expect(w.find('.alert-warning').text()).toContain('end date has passed')
    })
  })

  describe('28-day limit', () => {
    it('rejects a sprint longer than 28 days on create, saying why and giving the latest end date', async () => {
      const w = mount(SprintForm, { props: { projectKey: 'P' } })
      await w.find('input[type="text"]').setValue('Too long')
      const dateInputs = w.findAll('input[type="date"]')
      await dateInputs[0]!.setValue(daysFromToday(1))
      await dateInputs[1]!.setValue(daysFromToday(30)) // 29 days
      await w.find('form').trigger('submit')
      await flushPromises()
      expect(mockCreateSprint).not.toHaveBeenCalled()
      expect(w.text()).toContain("A sprint can't be longer than 28 days (4 weeks)")
      expect(w.text()).toContain('latest end date')
    })

    it('accepts exactly 28 days and sets the picker maximum from the start date', async () => {
      mockCreateSprint.mockResolvedValueOnce(editSprint)
      const w = mount(SprintForm, { props: { projectKey: 'P' } })
      await w.find('input[type="text"]').setValue('Max')
      const dateInputs = w.findAll('input[type="date"]')
      await dateInputs[0]!.setValue(daysFromToday(1))
      await dateInputs[1]!.setValue(daysFromToday(29))
      expect(dateInputs[1]!.attributes('max')).toBe(daysFromToday(29))
      await w.find('form').trigger('submit')
      await flushPromises()
      expect(mockCreateSprint).toHaveBeenCalled()
    })

    it('applies when editing a planning sprint', async () => {
      const w = mount(SprintForm, { props: { projectKey: 'P', sprint: editSprint } })
      await w.findAll('input[type="date"]')[1]!.setValue(daysFromToday(7 + 40))
      await w.find('form').trigger('submit')
      await flushPromises()
      expect(mockUpdateSprint).not.toHaveBeenCalled()
      expect(w.text()).toContain("can't be longer than 28 days")
    })
  })

  describe('which fields can be edited', () => {
    const base = { projectKey: 'P' }
    it('planning: every field is editable', () => {
      const w = mount(SprintForm, { props: { ...base, sprint: editSprint } })
      expect(w.findAll('input, textarea').every(i => i.attributes('disabled') === undefined)).toBe(true)
    })

    it('active: only the end date is editable, with a note saying so', () => {
      const w = mount(SprintForm, { props: { ...base, sprint: { ...editSprint, status: 'active' } } })
      const dates = w.findAll('input[type="date"]')
      expect(w.find('input[type="text"]').attributes('disabled')).toBeDefined()
      expect(w.find('textarea').attributes('disabled')).toBeDefined()
      expect(dates[0]!.attributes('disabled')).toBeDefined()
      expect(dates[1]!.attributes('disabled')).toBeUndefined()
      expect(w.text()).toContain('only the end date can be changed')
    })

    it('completed: nothing is editable and Save is disabled', () => {
      const w = mount(SprintForm, { props: { ...base, sprint: { ...editSprint, status: 'completed' } } })
      expect(w.findAll('input, textarea').every(i => i.attributes('disabled') !== undefined)).toBe(true)
      expect(w.find('button[type="submit"]').attributes('disabled')).toBeDefined()
      expect(w.text()).toContain('cannot be changed')
    })
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

// @vitest-environment happy-dom
import { describe, expect, test, vi, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import IssueSearchBar from '../IssueSearchBar.vue'

describe('IssueSearchBar', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  test('emits search event with q when text is typed (debounced 300ms)', async () => {
    const wrapper = mount(IssueSearchBar)
    const input = wrapper.find('input[type="text"]')
    await input.setValue('login bug')
    // Before 300ms — no event yet
    vi.advanceTimersByTime(299)
    await nextTick()
    expect(wrapper.emitted('search')).toBeFalsy()
    // After debounce fires
    vi.advanceTimersByTime(1)
    await nextTick()
    const events = wrapper.emitted('search') as [{ q?: string }][][]
    expect(events).toHaveLength(1)
    expect(events[0]![0]?.q).toBe('login bug')
  })

  test('emits search event when priority dropdown changes', async () => {
    const wrapper = mount(IssueSearchBar)
    // Priority is the 3rd select (index 2)
    const prioritySelect = wrapper.findAll('select')[2]!
    await prioritySelect.setValue('high')
    vi.advanceTimersByTime(300)
    await nextTick()
    const events = wrapper.emitted('search') as [{ priority?: string }][][]
    expect(events?.length).toBeGreaterThan(0)
    const last = events[events.length - 1]![0]!
    expect(last.priority).toBe('high')
  })

  test('clearing individual chip emits search without that filter', async () => {
    const wrapper = mount(IssueSearchBar)
    // Type something to show chip
    await wrapper.find('input[type="text"]').setValue('bug')
    vi.advanceTimersByTime(300)
    await nextTick()
    // Click ✕ button on the q chip (first badge button)
    const chipBtn = wrapper.find('.badge button')
    await chipBtn.trigger('click')
    vi.advanceTimersByTime(300)
    await nextTick()
    const events = wrapper.emitted('search') as [{ q?: string }][][]
    const last = events[events.length - 1]![0]!
    expect(last.q).toBeUndefined()
  })

  test('clearAll emits search with all filters undefined', async () => {
    const wrapper = mount(IssueSearchBar)
    // Set a filter so clearAll button appears
    await wrapper.find('input[type="text"]').setValue('test')
    vi.advanceTimersByTime(300)
    await nextTick()
    // Click "Clear all" button
    const clearAllBtn = wrapper.find('button.btn-ghost')
    await clearAllBtn.trigger('click')
    vi.advanceTimersByTime(300)
    await nextTick()
    const events = wrapper.emitted('search') as [{ q?: string; type?: string; status?: string; priority?: string }][][]
    const last = events[events.length - 1]![0]!
    expect(last.q).toBeUndefined()
    expect(last.type).toBeUndefined()
    expect(last.status).toBeUndefined()
    expect(last.priority).toBeUndefined()
  })
})

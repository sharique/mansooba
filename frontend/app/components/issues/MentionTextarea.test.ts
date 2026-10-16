// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import MentionTextarea from './MentionTextarea.vue'
import type { MemberResponse } from '~/types/domain.types'

const mem = (id: number, name: string, username: string): MemberResponse =>
  ({ user_id: id, name, email: `${username}@example.com`, username, role: 'member', is_active: true })
const members = [mem(1, 'Me Myself', 'me'), mem(2, 'Sam Smith', 'sam'), mem(3, 'Samira Khan', 'samira'), mem(4, 'Zoe Adams', 'zoe')]

function make(value = '', props: Record<string, unknown> = {}) {
  const w = mount(MentionTextarea, {
    props: {
      modelValue: value,
      members,
      currentUserId: 1,
      'onUpdate:modelValue': (v: string) => w.setProps({ modelValue: v }),
      ...props,
    },
    attachTo: document.body,
  })
  return w
}

async function type(w: ReturnType<typeof make>, text: string) {
  const ta = w.find('textarea')
  const el = ta.element as HTMLTextAreaElement
  el.value = text
  el.setSelectionRange(text.length, text.length)
  await ta.trigger('input')
}

describe('MentionTextarea', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => { vi.useRealTimers(); document.body.innerHTML = '' })

  it('lists other members with name over @username and no email', async () => {
    const w = make()
    await type(w, '@')
    const options = w.findAll('[role="option"]')
    expect(options.map(o => o.text())).toEqual([
      expect.stringContaining('Sam Smith'), expect.stringContaining('Samira Khan'), expect.stringContaining('Zoe Adams'),
    ])
    expect(options[0]!.text()).toContain('@sam')
    expect(w.html()).not.toContain('example.com')
    expect(w.html()).not.toContain('Me Myself')
  })

  it('narrows as you type and says when nothing matches', async () => {
    const w = make()
    await type(w, '@zo')
    expect(w.findAll('[role="option"]')).toHaveLength(1)
    await type(w, '@qq')
    expect(w.findAll('[role="option"]')).toHaveLength(0)
    expect(w.text()).toContain('No matching members')
  })

  it('arrow keys then Enter insert @username and a space', async () => {
    const w = make()
    await type(w, 'hi @sa')
    await w.find('textarea').trigger('keydown', { key: 'ArrowDown' })
    await w.find('textarea').trigger('keydown', { key: 'Enter' })
    expect(w.props('modelValue')).toBe('hi @samira ')
    expect(w.find('[role="listbox"]').exists()).toBe(false)
  })

  it('a click inserts that entry', async () => {
    const w = make()
    await type(w, '@')
    await w.findAll('[role="option"]')[2]!.trigger('mousedown')
    expect(w.props('modelValue')).toBe('@zoe ')
  })

  it('Escape closes the list and keeps the text', async () => {
    const w = make()
    await type(w, '@sa')
    await w.find('textarea').trigger('keydown', { key: 'Escape' })
    expect(w.find('[role="listbox"]').exists()).toBe(false)
    expect(w.props('modelValue')).toBe('@sa')
  })

  it('Enter with the list closed is left to the textarea', async () => {
    const w = make()
    await type(w, 'plain')
    const ev = new KeyboardEvent('keydown', { key: 'Enter', cancelable: true })
    w.find('textarea').element.dispatchEvent(ev)
    expect(ev.defaultPrevented).toBe(false)
  })

  it('exposes combobox / listbox ARIA with the active option', async () => {
    const w = make()
    const ta = w.find('textarea')
    expect(ta.attributes('role')).toBe('combobox')
    expect(ta.attributes('aria-expanded')).toBe('false')
    await type(w, '@sa')
    expect(ta.attributes('aria-expanded')).toBe('true')
    expect(ta.attributes('aria-autocomplete')).toBe('list')
    const list = w.find('[role="listbox"]')
    expect(ta.attributes('aria-controls')).toBe(list.attributes('id'))
    const first = w.findAll('[role="option"]')[0]!
    expect(first.attributes('aria-selected')).toBe('true')
    expect(ta.attributes('aria-activedescendant')).toBe(first.attributes('id'))
  })

  it('warns about a non-member about 400 ms after typing pauses', async () => {
    const w = make()
    await type(w, 'ping @lee ')
    expect(w.find('[role="status"]').text()).toBe('')
    vi.advanceTimersByTime(400)
    await w.vm.$nextTick()
    expect(w.find('[role="status"]').text()).toContain('@lee is not a member of this project')
    await type(w, 'ping @sam ')
    vi.advanceTimersByTime(400)
    await w.vm.$nextTick()
    expect(w.find('[role="status"]').text()).toBe('')
  })

  it('never warns for members, emails or when members are not loaded', async () => {
    const w = make()
    await type(w, '@sam mail a@b.com')
    vi.advanceTimersByTime(400)
    await w.vm.$nextTick()
    expect(w.find('[role="status"]').text()).toBe('')
    const none = make('', { members: [] })
    await type(none, '@lee')
    vi.advanceTimersByTime(400)
    await none.vm.$nextTick()
    expect(none.find('[role="status"]').text()).toBe('')
  })

  it('passes rows and placeholder through', () => {
    const w = make('', { rows: 5, placeholder: 'Say it' })
    expect(w.find('textarea').attributes('rows')).toBe('5')
    expect(w.find('textarea').attributes('placeholder')).toBe('Say it')
  })
})

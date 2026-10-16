// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { useMentionAutocomplete } from './useMentionAutocomplete'
import type { MemberResponse } from '~/types/domain.types'

const mem = (id: number, name: string, username: string): MemberResponse =>
  ({ user_id: id, name, email: `${username}@example.com`, username, role: 'member', is_active: true })

const key = (k: string) => ({ key: k, preventDefault: vi.fn() }) as unknown as KeyboardEvent

function setup(members: MemberResponse[] = [mem(1, 'Me', 'me'), mem(2, 'Sam Smith', 'sam'), mem(3, 'Samira Khan', 'samira'), mem(4, 'Zoe', 'zoe')]) {
  const text = ref('')
  const memberRef = ref(members)
  const ac = useMentionAutocomplete({ text, members: memberRef, currentUserId: ref(1), warnDelay: 400 })
  const type = (t: string, caret = t.length) => { text.value = t; ac.onInput(caret) }
  return { text, memberRef, ac, type }
}

describe('useMentionAutocomplete', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('opens when a query appears and closes when it goes away', () => {
    const { ac, type } = setup()
    type('hi')
    expect(ac.open.value).toBe(false)
    type('hi @sa')
    expect(ac.open.value).toBe(true)
    expect(ac.suggestions.value.map(m => m.username)).toEqual(['sam', 'samira'])
    type('hi @sa ')
    expect(ac.open.value).toBe(false)
  })

  it('never offers the current user and never opens without members', () => {
    const { ac, type, memberRef } = setup()
    type('@')
    expect(ac.suggestions.value.map(m => m.username)).not.toContain('me')
    memberRef.value = []
    type('@s')
    expect(ac.open.value).toBe(false)
  })

  it('moves with ArrowDown/ArrowUp and wraps', () => {
    const { ac, type } = setup()
    type('@sa')
    expect(ac.activeIndex.value).toBe(0)
    const down = key('ArrowDown')
    expect(ac.onKeydown(down)).toBe(true)
    expect(down.preventDefault).toHaveBeenCalled()
    expect(ac.activeIndex.value).toBe(1)
    ac.onKeydown(key('ArrowDown'))
    expect(ac.activeIndex.value).toBe(0)
    ac.onKeydown(key('ArrowUp'))
    expect(ac.activeIndex.value).toBe(1)
  })

  it('Enter and Tab select only while open', () => {
    const { ac, type, text } = setup()
    type('@sa')
    ac.onKeydown(key('ArrowDown'))
    const enter = key('Enter')
    expect(ac.onKeydown(enter)).toBe(true)
    expect(enter.preventDefault).toHaveBeenCalled()
    expect(text.value).toBe('@samira ')
    expect(ac.open.value).toBe(false)

    const again = key('Enter')
    expect(ac.onKeydown(again)).toBe(false)
    expect(again.preventDefault).not.toHaveBeenCalled()
    expect(ac.onKeydown(key('Tab'))).toBe(false)

    type('@zo')
    expect(ac.onKeydown(key('Tab'))).toBe(true)
    expect(text.value).toBe('@zoe ')
  })

  it('select reports the new caret', () => {
    const { ac, type } = setup()
    type('hi @sa there', 6)
    const res = ac.select(ac.suggestions.value[0]!)
    expect(res).toEqual({ text: 'hi @sam  there', caret: 8 })
  })

  it('Escape closes and keeps the text', () => {
    const { ac, type, text } = setup()
    type('@sa')
    expect(ac.onKeydown(key('Escape'))).toBe(true)
    expect(ac.open.value).toBe(false)
    expect(text.value).toBe('@sa')
    expect(ac.onKeydown(key('Escape'))).toBe(false)
  })

  it('resets the active entry when the filter changes', () => {
    const { ac, type } = setup()
    type('@sa')
    ac.onKeydown(key('ArrowDown'))
    type('@sam')
    expect(ac.activeIndex.value).toBe(0)
  })

  it('open with no match is still open but empty', () => {
    const { ac, type } = setup()
    type('@qq')
    expect(ac.open.value).toBe(true)
    expect(ac.suggestions.value).toEqual([])
    expect(ac.onKeydown(key('Enter'))).toBe(false)
  })

  it('lists unknown mentions about 400 ms after typing pauses', async () => {
    const { ac, type } = setup()
    type('hi @lee')
    expect(ac.unknown.value).toEqual([])
    vi.advanceTimersByTime(399)
    await nextTick()
    expect(ac.unknown.value).toEqual([])
    vi.advanceTimersByTime(1)
    await nextTick()
    expect(ac.unknown.value).toEqual(['lee'])
    type('hi @sam')
    vi.advanceTimersByTime(400)
    await nextTick()
    expect(ac.unknown.value).toEqual([])
  })

  it('clears the warning when the text is emptied (after a post)', async () => {
    const { ac, type, text } = setup()
    type('@lee')
    vi.advanceTimersByTime(400)
    expect(ac.unknown.value).toEqual(['lee'])
    text.value = ''
    ac.onInput(0)
    expect(ac.unknown.value).toEqual([])
  })
})

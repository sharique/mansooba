// @vitest-environment happy-dom
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { describe, expect, it } from 'vitest'
import { applySuggestion, filterSuggestions, findMentionQuery, findUnknownMentions, renderMentions } from './useMentions'
import type { MemberResponse } from '~/types/domain.types'

const m = (user_id: number, name: string, username: string, email = `${username}@example.com`): MemberResponse =>
  ({ user_id, name, email, username, role: 'member', is_active: true })

const members: MemberResponse[] = [
  m(1, 'Priya Nair', 'priya_nair', 'priya.nair@example.com'),
  m(2, 'Sam', 'sam'),
  m(3, '<b>Evil</b>', 'evil'),
]

const span = (username: string, name: string) => `<span class="mention" title="@${username}">@${name}</span>`

describe('renderMentions (by username)', () => {
  it('replaces @username with the display name and keeps the username in the title', () => {
    expect(renderMentions('hi @sam please look', members)).toBe(`hi ${span('sam', 'Sam')} please look`)
  })

  it('supports underscores and digits in the username', () => {
    expect(renderMentions('@priya_nair ok', members)).toBe(`${span('priya_nair', 'Priya Nair')} ok`)
  })

  it('matches without regard to letter case, like the server', () => {
    expect(renderMentions('@Sam @SAM', members)).toBe(`${span('sam', 'Sam')} ${span('sam', 'Sam')}`)
  })

  it('handles several and repeated mentions', () => {
    expect(renderMentions('@sam and @priya_nair and @sam', members))
      .toBe(`${span('sam', 'Sam')} and ${span('priya_nair', 'Priya Nair')} and ${span('sam', 'Sam')}`)
  })

  it('leaves unknown names and non-members as typed', () => {
    expect(renderMentions('hello @nobody', members)).toBe('hello @nobody')
  })

  it('a trailing sentence dot is not part of the username', () => {
    expect(renderMentions('thanks @sam.', members)).toBe(`thanks ${span('sam', 'Sam')}.`)
  })

  it('works after punctuation and markdown markers, as on the server', () => {
    expect(renderMentions('(@sam)', members)).toBe(`(${span('sam', 'Sam')})`)
    expect(renderMentions('**@sam**', members)).toBe(`**${span('sam', 'Sam')}**`)
    expect(renderMentions('> @sam quoted', members)).toBe(`> ${span('sam', 'Sam')} quoted`)
    expect(renderMentions('thanks,@sam', members)).toBe(`thanks,${span('sam', 'Sam')}`)
    expect(renderMentions('line\n@sam', members)).toBe(`line\n${span('sam', 'Sam')}`)
  })

  it('does not treat an email address or a word-internal @ as a mention', () => {
    expect(renderMentions('mail a@sam.com', members)).toBe('mail a@sam.com')
    expect(renderMentions('a.b@sam', members)).toBe('a.b@sam')
    expect(renderMentions('x_@sam', members)).toBe('x_@sam')
  })

  it('no longer matches the start of an email address', () => {
    // priya.nair@example.com belongs to the username priya_nair, not "priya.nair".
    expect(renderMentions('@priya.nair', members)).toBe('@priya.nair')
  })

  it('escapes HTML in display names', () => {
    expect(renderMentions('@evil', members)).toBe(span('evil', '&lt;b&gt;Evil&lt;/b&gt;'))
  })

  it('returns the body unchanged without members', () => {
    expect(renderMentions('@sam', [])).toBe('@sam')
  })

  it('ignores members that have no username yet', () => {
    expect(renderMentions('@sam', [{ ...m(9, 'Legacy', ''), username: '' }])).toBe('@sam')
  })

  it('does not touch mentions inside code', () => {
    expect(renderMentions('use `@sam` here', members)).toBe('use `@sam` here')
    expect(renderMentions('```\n@sam\n```', members)).toBe('```\n@sam\n```')
  })

  it('survives markdown + DOMPurify with the span and title intact', () => {
    const html = DOMPurify.sanitize(marked.parse(renderMentions('hi @sam', members)) as string)
    expect(html).toContain('<span class="mention" title="@sam">@Sam</span>')
  })
})

// ── typing a mention ─────────────────────────────────────────────────────────

describe('findMentionQuery', () => {
  const q = (text: string, caret = text.length) => findMentionQuery(text, caret)

  it('finds the fragment after an @ at the start of the text', () => {
    expect(q('@')).toEqual({ start: 0, query: '' })
    expect(q('@sa')).toEqual({ start: 0, query: 'sa' })
  })

  it('finds it after a space, a newline and markdown or punctuation', () => {
    expect(q('hi @sa')).toEqual({ start: 3, query: 'sa' })
    expect(q('line\n@sa')).toEqual({ start: 5, query: 'sa' })
    expect(q('(@sa')).toEqual({ start: 1, query: 'sa' })
    expect(q('**@sa')).toEqual({ start: 2, query: 'sa' })
    expect(q('> @sa')).toEqual({ start: 2, query: 'sa' })
    expect(q('thanks,@sa')).toEqual({ start: 7, query: 'sa' })
  })

  it('allows letters, digits and underscores in the query', () => {
    expect(q('@sam_2')).toEqual({ start: 0, query: 'sam_2' })
  })

  it('does not open inside a word or an email address', () => {
    expect(q('a@b')).toBeNull()
    expect(q('a.b@c')).toBeNull()
    expect(q('x_@y')).toBeNull()
    expect(q('mail me at a@')).toBeNull()
  })

  it('works from the caret, not the end of the text', () => {
    expect(q('hello @sa and more', 9)).toEqual({ start: 6, query: 'sa' })
    expect(q('hello @sa and more', 6)).toBeNull() // caret before the @
    expect(q('hello @sa and more', 12)).toBeNull() // a space after the query
  })

  it('is null for empty text or no @', () => {
    expect(q('')).toBeNull()
    expect(q('no mention here')).toBeNull()
  })
})

describe('filterSuggestions', () => {
  const mem = (id: number, name: string, username: string, extra: Partial<MemberResponse> = {}): MemberResponse =>
    ({ user_id: id, name, email: `${username}@example.com`, username, role: 'member', is_active: true, ...extra })
  const all = [
    mem(1, 'Zoe Adams', 'zoe'),
    mem(2, 'Sam Smith', 'sam'),
    mem(3, 'Priya Nair', 'priya_nair'),
    mem(4, 'Samira Khan', 'samira'),
    mem(5, 'Former Colleague', 'former', { is_active: false }),
    mem(6, 'No Username Yet', ''),
    mem(7, 'The Author', 'author'),
  ]

  it('matches display name or username, ignoring case', () => {
    expect(filterSuggestions(all, 'PRIYA', 7).map(m => m.user_id)).toEqual([3])
    expect(filterSuggestions(all, 'smith', 7).map(m => m.user_id)).toEqual([2])
  })

  it('puts prefix matches first, then the rest, each alphabetical by name', () => {
    expect(filterSuggestions(all, 'sam', 7).map(m => m.user_id)).toEqual([2, 4])
  })

  it('lists non-prefix matches after prefix matches', () => {
    // "air" is inside "Priya Nair" only as a substring; "pri" is a prefix of Priya.
    expect(filterSuggestions(all, 'ai', 7).map(m => m.user_id)).toEqual([3])
    expect(filterSuggestions(all, 'a', 7).map(m => m.user_id)).toEqual([3, 2, 4, 1])
  })

  it('returns everyone else for an empty query', () => {
    expect(filterSuggestions(all, '', 7).map(m => m.user_id)).toEqual([3, 2, 4, 1])
  })

  it('leaves out the current user, deactivated members and members without a username', () => {
    const ids = filterSuggestions(all, '', 7).map(m => m.user_id)
    expect(ids).not.toContain(7)
    expect(ids).not.toContain(5)
    expect(ids).not.toContain(6)
  })

  it('is empty when nothing matches or there are no members', () => {
    expect(filterSuggestions(all, 'zzzz', 7)).toEqual([])
    expect(filterSuggestions([], '', 7)).toEqual([])
  })

  it('stays fast with hundreds of members', () => {
    const many = Array.from({ length: 500 }, (_, i) => mem(i + 10, `Person ${i}`, `person_${i}`))
    const t0 = performance.now()
    for (let i = 0; i < 20; i++) filterSuggestions(many, 'per', 1)
    expect(performance.now() - t0).toBeLessThan(250)
  })
})

describe('applySuggestion', () => {
  it('replaces the typed fragment with @username and a space, caret after it', () => {
    const text = 'hi @sa please'
    const res = applySuggestion(text, { start: 3, query: 'sa' }, 'sam')
    expect(res.text).toBe('hi @sam  please')
    expect(res.caret).toBe('hi @sam '.length)
  })

  it('works at the end of the text and for an empty query', () => {
    expect(applySuggestion('@', { start: 0, query: '' }, 'sam')).toEqual({ text: '@sam ', caret: 5 })
    expect(applySuggestion('hello @', { start: 6, query: '' }, 'priya_nair')).toEqual({ text: 'hello @priya_nair ', caret: 18 })
  })

  it('keeps the text after the caret intact', () => {
    const res = applySuggestion('(@sa) ok', { start: 1, query: 'sa' }, 'sam')
    expect(res.text).toBe('(@sam ) ok')
  })
})

describe('findUnknownMentions', () => {
  const members: MemberResponse[] = [
    { user_id: 1, name: 'Sam', email: 'sam@example.com', username: 'sam', role: 'member', is_active: true },
    { user_id: 2, name: 'Priya', email: 'p@example.com', username: 'priya_nair', role: 'member', is_active: true },
  ]

  it('lists mentions that are not a member, once each, as typed', () => {
    expect(findUnknownMentions('hi @lee and @Lee and @sam', members)).toEqual(['lee'])
    expect(findUnknownMentions('@lee @kim', members)).toEqual(['lee', 'kim'])
  })

  it('is case-insensitive for members', () => {
    expect(findUnknownMentions('@SAM @Priya_Nair', members)).toEqual([])
  })

  it('ignores email addresses and word-internal @', () => {
    expect(findUnknownMentions('mail a@b.com or x.y@z', members)).toEqual([])
  })

  it('ignores code spans and fences', () => {
    expect(findUnknownMentions('use `@lee` here', members)).toEqual([])
    expect(findUnknownMentions('```\n@lee\n```', members)).toEqual([])
  })

  it('works after punctuation and markdown markers', () => {
    expect(findUnknownMentions('(@lee) **@kim** > @amy', members)).toEqual(['lee', 'kim', 'amy'])
  })

  it('says nothing when members are not loaded', () => {
    expect(findUnknownMentions('@lee', [])).toEqual([])
  })

  it('says nothing about a trailing dot', () => {
    expect(findUnknownMentions('thanks @sam.', members)).toEqual([])
  })
})

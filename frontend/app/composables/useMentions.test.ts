// @vitest-environment happy-dom
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { describe, expect, it } from 'vitest'
import { renderMentions } from './useMentions'
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

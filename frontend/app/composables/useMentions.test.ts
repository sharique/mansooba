// @vitest-environment happy-dom
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { describe, expect, it } from 'vitest'
import { renderMentions } from './useMentions'
import type { MemberResponse } from '~/types/domain.types'

const members: MemberResponse[] = [
  { user_id: 1, name: 'Priya Nair', email: 'priya.nair@example.com', role: 'member' },
  { user_id: 2, name: 'Sam', email: 'sam@example.com', role: 'admin' },
  { user_id: 3, name: '<b>Evil</b>', email: 'evil@example.com', role: 'member' },
]

const span = (handle: string, name: string) => `<span class="mention" title="@${handle}">@${name}</span>`

describe('renderMentions', () => {
  it('replaces @handle with the member display name and keeps the handle in the title', () => {
    expect(renderMentions('hi @sam please look', members)).toBe(`hi ${span('sam', 'Sam')} please look`)
  })

  it('supports dots in the handle (email local part)', () => {
    expect(renderMentions('@priya.nair ok', members)).toBe(`${span('priya.nair', 'Priya Nair')} ok`)
  })

  it('handles several and repeated mentions', () => {
    expect(renderMentions('@sam and @priya.nair and @sam', members))
      .toBe(`${span('sam', 'Sam')} and ${span('priya.nair', 'Priya Nair')} and ${span('sam', 'Sam')}`)
  })

  it('leaves unknown handles and non-members as typed', () => {
    expect(renderMentions('hello @nobody', members)).toBe('hello @nobody')
  })

  it('is exact and case-sensitive', () => {
    expect(renderMentions('@Sam @SAM', members)).toBe('@Sam @SAM')
  })

  it('mirrors the server pattern: a trailing dot is part of the handle and matches nobody', () => {
    expect(renderMentions('thanks @sam.', members)).toBe('thanks @sam.')
  })

  it('does not treat an email address as a mention of a member', () => {
    expect(renderMentions('mail a@sam.com', members)).toBe('mail a@sam.com')
  })

  it('escapes HTML in display names', () => {
    expect(renderMentions('@evil', members)).toBe(span('evil', '&lt;b&gt;Evil&lt;/b&gt;'))
  })

  it('consumes the whole address when the full email was typed', () => {
    expect(renderMentions('@sam@example.com it looks good.', members)).toBe(`${span('sam', 'Sam')} it looks good.`)
  })

  it('keeps a trailing sentence dot after a full-email mention', () => {
    expect(renderMentions('thanks @sam@example.com.', members)).toBe(`thanks ${span('sam', 'Sam')}.`)
  })

  it('matches the domain without regard to case', () => {
    expect(renderMentions('@sam@Example.COM ok', members)).toBe(`${span('sam', 'Sam')} ok`)
  })

  it('leaves a different domain text alone (only the handle is substituted)', () => {
    expect(renderMentions('@sam@other.org ok', members)).toBe(`${span('sam', 'Sam')}@other.org ok`)
  })

  it('does not swallow text that merely starts like the domain', () => {
    expect(renderMentions('@sam@example.community', members)).toBe(`${span('sam', 'Sam')}@example.community`)
  })

  it('returns the body unchanged without members', () => {
    expect(renderMentions('@sam', [])).toBe('@sam')
  })

  it('does not touch mentions inside code', () => {
    expect(renderMentions('use `@sam` here', members)).toBe('use `@sam` here')
    expect(renderMentions('```\n@sam\n```', members)).toBe('```\n@sam\n```')
  })

  it('survives markdown + DOMPurify with the span and title intact', () => {
    const html = DOMPurify.sanitize(marked.parse(renderMentions('hi @sam', members)) as string)
    expect(html).toContain('<span class="mention" title="@sam">@Sam</span>')
  })

  it('cannot smuggle markup through a name', () => {
    const html = DOMPurify.sanitize(marked.parse(renderMentions('@evil', members)) as string)
    expect(html).not.toContain('<b>')
  })
})

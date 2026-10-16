import type { MemberResponse } from '~/types/domain.types'

// Same rule the server uses to find mentions (comment_service.go, ADR-038): "@"
// plus a username (letters, digits, underscore). The "@" must not follow a
// letter, digit, underscore, dot or another "@", so "(@sam)", "**@sam**" and
// "> @sam" are mentions while "a@b.com" and "a.b@c" are not. The character
// before the "@" is part of the match and is put back.
const MENTION = /(^|[^A-Za-z0-9_.@])@([A-Za-z0-9_]+)/g

// Code spans and fences are left alone so `@sam` in code stays literal.
const CODE = /(```[\s\S]*?```|`[^`\n]*`)/

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

/**
 * Display-only: replaces `@username` with the project member's display name
 * (the username stays in the title). Matching ignores letter case, like the
 * server. Only project members' names are available to the page, so a mention
 * of anyone else, or of nobody, is left as typed. Stored text and who gets
 * notified are unchanged.
 *
 * The result goes through `useMarkdown`, whose DOMPurify pass still sanitizes it
 * (ADR-022); names are escaped here as well.
 */
export function renderMentions(body: string, members: MemberResponse[]): string {
  if (!members.length || !body.includes('@')) return body
  const byUsername = new Map<string, MemberResponse>()
  for (const m of members) {
    if (m.username) byUsername.set(m.username.toLowerCase(), m)
  }
  return body
    .split(CODE)
    .map((part, i) => i % 2 === 1
      ? part
      : part.replace(MENTION, (whole, before: string, handle: string) => {
          const member = byUsername.get(handle.toLowerCase())
          if (!member) return whole
          return `${before}<span class="mention" title="@${escapeHtml(member.username)}">@${escapeHtml(member.name)}</span>`
        }))
    .join('')
}

// ── typing a mention ─────────────────────────────────────────────────────────

export interface MentionQuery { start: number; query: string }

/**
 * The "@fragment" the caret is in, or null. The "@" must open a mention by the
 * same boundary rule as `MENTION` (not after a letter, digit, underscore, dot or
 * "@"), and only username characters may sit between it and the caret.
 */
export function findMentionQuery(text: string, caret: number): MentionQuery | null {
  const head = text.slice(0, caret)
  const m = /(^|[^A-Za-z0-9_.@])@([A-Za-z0-9_]*)$/.exec(head)
  if (!m) return null
  return { start: head.length - m[2]!.length - 1, query: m[2]! }
}

/**
 * Members to offer for `query`: active members with a username, minus the
 * current user. Matches display name or username (any case); names or
 * usernames that start with the query come first, each group by name.
 */
export function filterSuggestions(members: MemberResponse[], query: string, currentUserId: number): MemberResponse[] {
  const q = query.toLowerCase()
  const prefix: MemberResponse[] = []
  const rest: MemberResponse[] = []
  for (const m of members) {
    if (!m.is_active || !m.username || m.user_id === currentUserId) continue
    const name = m.name.toLowerCase()
    const user = m.username.toLowerCase()
    if (name.startsWith(q) || user.startsWith(q)) prefix.push(m)
    else if (name.includes(q) || user.includes(q)) rest.push(m)
  }
  const byName = (a: MemberResponse, b: MemberResponse) => a.name.localeCompare(b.name)
  return [...prefix.sort(byName), ...rest.sort(byName)]
}

/** Replaces the typed fragment with "@username " and returns the new caret. */
export function applySuggestion(text: string, mq: MentionQuery, username: string): { text: string; caret: number } {
  const before = text.slice(0, mq.start)
  const after = text.slice(mq.start + 1 + mq.query.length)
  const inserted = `@${username} `
  return { text: before + inserted + after, caret: before.length + inserted.length }
}

/**
 * Handles in `body` that are not a loaded project member, once each (first
 * spelling kept), in order. Code spans and fences are skipped. With no members
 * loaded it says nothing rather than warning about everyone.
 */
export function findUnknownMentions(body: string, members: MemberResponse[]): string[] {
  if (!members.length || !body.includes('@')) return []
  const known = new Set(members.map(m => m.username.toLowerCase()))
  const seen = new Set<string>()
  const out: string[] = []
  body.split(CODE).forEach((part, i) => {
    if (i % 2 === 1) return
    for (const m of part.matchAll(MENTION)) {
      const handle = m[2]!
      const key = handle.toLowerCase()
      if (known.has(key) || seen.has(key)) continue
      seen.add(key)
      out.push(handle)
    }
  })
  return out
}

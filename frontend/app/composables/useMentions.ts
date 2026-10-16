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

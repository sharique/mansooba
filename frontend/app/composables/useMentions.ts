import type { MemberResponse } from '~/types/domain.types'

// Same pattern the server uses to find mentions (comment_service.go). No word
// boundary, so `a@b.com` yields the handle `b.com`, which matches no member.
// An optional `@domain` is captured too: people often type the full address
// (`@sam@example.com`); the server reads the handle as `sam` and ignores the rest.
const MENTION = /@([\w.]+)(@[\w.-]+)?/g

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
 * Display-only: replaces `@handle` with the project member's name when `handle`
 * is exactly (case-sensitive) their email local part. Stored text and who gets
 * notified are unchanged. Case-sensitive on purpose: the server matches with SQL
 * LIKE, case-insensitive on SQLite but not PostgreSQL, so exact matching can only
 * under-substitute, never name someone who was not notified.
 *
 * The result goes through `useMarkdown`, whose DOMPurify pass still sanitizes it
 * (ADR-022); names are escaped here as well.
 */
export function renderMentions(body: string, members: MemberResponse[]): string {
  if (!members.length || !body.includes('@')) return body
  const byHandle = new Map<string, MemberResponse>()
  for (const m of members) {
    const handle = m.email.split('@')[0]
    if (handle && !byHandle.has(handle)) byHandle.set(handle, m)
  }
  return body
    .split(CODE)
    .map((part, i) => i % 2 === 1
      ? part
      : part.replace(MENTION, (whole, handle: string, domainPart?: string) => {
          const member = byHandle.get(handle)
          if (!member) return whole
          const span = `<span class="mention" title="@${handle}">@${escapeHtml(member.name)}</span>`
          if (!domainPart) return span
          // Swallow the typed domain only when it is this member's own domain
          // (plus trailing sentence dots); otherwise leave it as typed.
          const own = `@${member.email.split('@')[1] ?? ''}`.toLowerCase()
          const lower = domainPart.toLowerCase()
          if (lower.startsWith(own) && /^\.*$/.test(domainPart.slice(own.length))) {
            return span + domainPart.slice(own.length)
          }
          return span + domainPart
        }))
    .join('')
}

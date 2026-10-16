// Username rules (feature 018, ADR-038). This is the TypeScript twin of
// backend/internal/domain/username.go; both are tested against the same
// vectors (backend/internal/domain/testdata/username_vectors.json), so the
// create-user forms' live pre-fill and the server's validation cannot drift.

export const USERNAME_MIN_LENGTH = 3
export const USERNAME_MAX_LENGTH = 30

export const RESERVED_USERNAMES: ReadonlySet<string> = new Set([
  'everyone', 'all', 'here', 'channel', 'admin', 'system', 'support', 'mansooba',
])

export const USERNAME_INVALID_MESSAGE = 'Usernames are 3 to 30 characters: letters, digits and underscores only.'
export const USERNAME_RESERVED_MESSAGE = 'That username is reserved.'

export interface UsernameProblem {
  reason: 'invalid' | 'reserved'
  message: string
}

export function normalizeUsername(s: string): string {
  return s.trim().toLowerCase()
}

/** null when the (normalised) name is allowed; otherwise why not. Does not check availability. */
export function validateUsername(s: string): UsernameProblem | null {
  const n = normalizeUsername(s)
  if (n.length < USERNAME_MIN_LENGTH || n.length > USERNAME_MAX_LENGTH || !/^[a-z0-9_]+$/.test(n)) {
    return { reason: 'invalid', message: USERNAME_INVALID_MESSAGE }
  }
  if (RESERVED_USERNAMES.has(n)) {
    return { reason: 'reserved', message: USERNAME_RESERVED_MESSAGE }
  }
  return null
}

/**
 * A valid, non-reserved candidate from an email address: the part before the
 * last "@", lower-cased, characters outside a-z, 0-9 and "_" removed, padded
 * with "user" when fewer than three remain, cut to 30, and numbered when
 * reserved ("admin" becomes "admin2"). Whether the name is taken is not known
 * here; the server adds a number for that when it generates one.
 */
export function deriveUsername(email: string): string {
  const at = email.lastIndexOf('@')
  const local = at >= 0 ? email.slice(0, at) : email
  let name = local.toLowerCase().replace(/[^a-z0-9_]/g, '')
  if (name.length < USERNAME_MIN_LENGTH) name = 'user' + name
  if (name.length > USERNAME_MAX_LENGTH) name = name.slice(0, USERNAME_MAX_LENGTH)
  if (RESERVED_USERNAMES.has(name)) name += '2'
  return name
}

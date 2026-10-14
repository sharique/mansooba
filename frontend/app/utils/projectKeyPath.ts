/**
 * Project keys in URLs are always lower-case (the API returns them that way, but a
 * seeded project is stored as `DEMO`, and users can type or paste any case).
 * Returns the corrected path, or null when it is already fine.
 * Query string and hash are preserved untouched.
 */
export function lowercaseProjectKeyPath(fullPath: string): string | null {
  const match = /^\/projects\/([^/?#]+)/.exec(fullPath)
  if (!match) return null
  const key = match[1]!
  const lower = key.toLowerCase()
  if (key === lower) return null
  return `/projects/${lower}${fullPath.slice(match[0].length)}`
}

import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import { deriveUsername, normalizeUsername, validateUsername } from './username'

// The Go tests read the same file (backend/internal/domain/username_test.go),
// so the form's live pre-fill and the server's rules cannot drift apart.
const file = resolve(dirname(fileURLToPath(import.meta.url)), '../../../backend/internal/domain/testdata/username_vectors.json')
const vectors = JSON.parse(readFileSync(file, 'utf8')) as {
  derive: { email: string; username: string }[]
  validate: { input: string; valid: boolean; normalised?: string; reason?: string }[]
}

describe('shared username vectors', () => {
  it('has cases to run', () => {
    expect(vectors.derive.length).toBeGreaterThan(10)
    expect(vectors.validate.length).toBeGreaterThan(10)
  })

  it.each(vectors.derive)('deriveUsername($email) = $username', ({ email, username }) => {
    expect(deriveUsername(email)).toBe(username)
  })

  it.each(vectors.validate)('validateUsername($input)', ({ input, valid, normalised, reason }) => {
    const result = validateUsername(input)
    if (valid) {
      expect(result).toBeNull()
      expect(normalizeUsername(input)).toBe(normalised)
    }
    else {
      expect(result?.reason).toBe(reason)
      expect(result?.message).toBeTruthy()
    }
  })
})

describe('deriveUsername', () => {
  it('always produces a valid username', () => {
    for (const e of ['', '@', 'a', '!!!@x', 'ünï@x', 'z'.repeat(80) + '@x', 'admin@x', 'all@x', 'x@y@z']) {
      expect(validateUsername(deriveUsername(e)), e).toBeNull()
    }
  })
})

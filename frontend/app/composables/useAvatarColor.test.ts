import { describe, expect, test } from 'vitest'
import { useAvatarColor } from './useAvatarColor'

describe('useAvatarColor', () => {
  test('returns a CSS color string', () => {
    const color = useAvatarColor(1)
    expect(typeof color).toBe('string')
    expect(color.length).toBeGreaterThan(0)
  })

  test('returns correct palette index for userId 0 (index 0)', () => {
    // palette has 8 entries; userId 0 % 8 === 0
    const color = useAvatarColor(0)
    expect(color).toBe('oklch(0.55 0.14 192)')
  })

  test('returns correct palette index for userId 8 (wraps to index 0)', () => {
    expect(useAvatarColor(8)).toBe(useAvatarColor(0))
  })

  test('wraps at palette boundary (userId 7 is last, userId 8 wraps to first)', () => {
    const color7 = useAvatarColor(7)
    const color8 = useAvatarColor(8)
    const color0 = useAvatarColor(0)
    expect(color7).not.toBe(color0)
    expect(color8).toBe(color0)
  })

  test('different userIds produce different colors within palette', () => {
    const colors = Array.from({ length: 8 }, (_, i) => useAvatarColor(i))
    const unique = new Set(colors)
    expect(unique.size).toBe(8)
  })

  test('returns string for large userId', () => {
    const color = useAvatarColor(999)
    expect(typeof color).toBe('string')
    expect(color).toBe(useAvatarColor(999 % 8))
  })
})

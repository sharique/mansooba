import { describe, expect, it } from 'vitest'
import { markDragEnd, wasJustDragged } from './dragGuard'

describe('dragGuard', () => {
  it('reports a drag that ended a moment ago, then forgets it', () => {
    markDragEnd(1000)
    expect(wasJustDragged(1050)).toBe(true)
    expect(wasJustDragged(1101)).toBe(false)
  })

  it('is false before any drag has happened', () => {
    markDragEnd(0)
    expect(wasJustDragged(5000)).toBe(false)
  })
})

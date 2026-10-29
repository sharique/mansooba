import { describe, expect, it } from 'vitest'
import shared from '../../../testdata/board-order-cases.json'
import { priorityRank, sortBoardIssues } from './boardOrder'

// The same table drives the Go board service test, so the server and the browser
// order a column identically.
interface Case { name: string; cards: { id: number; priority: string }[]; expected: number[] }
const cases = shared.cases as Case[]

describe('sortBoardIssues (shared cases)', () => {
  for (const c of cases) {
    it(c.name, () => {
      const sorted = sortBoardIssues(c.cards as never[])
      expect(sorted.map(i => (i as { id: number }).id)).toEqual(c.expected)
    })
  }
})

describe('sortBoardIssues', () => {
  it('returns a new array and leaves the input untouched', () => {
    const input = [{ id: 2, priority: 'low' }, { id: 1, priority: 'critical' }]
    const copy = [...input]
    const sorted = sortBoardIssues(input as never[])
    expect(sorted).not.toBe(input)
    expect(input).toEqual(copy)
  })
})

describe('priorityRank', () => {
  it('ranks critical first and unknown last', () => {
    expect(priorityRank('critical')).toBeLessThan(priorityRank('high'))
    expect(priorityRank('high')).toBeLessThan(priorityRank('medium'))
    expect(priorityRank('medium')).toBeLessThan(priorityRank('low'))
    expect(priorityRank('low')).toBeLessThan(priorityRank('weird'))
  })
})

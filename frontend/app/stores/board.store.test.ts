import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useBoardStore } from '~/stores/board.store'
import { boardService } from '~/services/board.service'
import { issuesService } from '~/services/issues.service'

vi.mock('~/services/board.service', () => ({
  boardService: { getBoard: vi.fn() },
}))

vi.mock('~/services/issues.service', () => ({
  issuesService: { update: vi.fn() },
}))

const board = vi.mocked(boardService)
const issues = vi.mocked(issuesService)

const issue = (id: number, status: string, priority = 'medium') =>
  ({ id, key: `P-${id}`, status, priority, project_id: 1, title: `Task ${id}`, locked: false }) as never

const STATUSES = ['todo', 'in_progress', 'in_review', 'done']
const boardOf = (issues: Record<string, unknown[]> = {}) => ({
  columns: STATUSES.map(status => ({ status, issues: issues[status] ?? [] })),
}) as never

describe('useBoardStore load', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.resetAllMocks()
  })

  it('fills the four columns in order', async () => {
    board.getBoard.mockResolvedValue(boardOf({ todo: [issue(1, 'todo')], done: [issue(2, 'done')] }))
    const store = useBoardStore()
    await store.load('P')
    expect(store.columns.map(c => c.status)).toEqual(STATUSES)
    expect(store.columns[0]!.issues).toHaveLength(1)
    expect(store.columns[3]!.issues).toHaveLength(1)
  })

  it('sorts every column by priority then id, whatever order the server sent', async () => {
    board.getBoard.mockResolvedValue(boardOf({
      todo: [issue(5, 'todo', 'low'), issue(2, 'todo', 'critical'), issue(9, 'todo', 'medium'), issue(1, 'todo', 'high')],
    }))
    const store = useBoardStore()
    await store.load('P')
    expect(store.columns[0]!.issues.map(i => i.id)).toEqual([2, 1, 9, 5])
  })

  it('is loading until the first answer arrives', async () => {
    let resolve!: (v: never) => void
    board.getBoard.mockReturnValue(new Promise(r => { resolve = r as never }))
    const store = useBoardStore()
    expect(store.loading).toBe(true)
    const pending = store.load('P')
    expect(store.loading).toBe(true)
    resolve(boardOf())
    await pending
    expect(store.loading).toBe(false)
    expect(store.error).toBeNull()
  })

  it('a failed load sets the error and leaves the columns empty', async () => {
    board.getBoard.mockRejectedValue({ data: { message: 'forbidden' } })
    const store = useBoardStore()
    await store.load('P')
    expect(store.error).toBe('forbidden')
    expect(store.columns).toEqual([])
    expect(store.loading).toBe(false)
  })
})

// A promise the test settles by hand, to look at the board while a save is in flight.
function deferred<T = void>() {
  let resolve!: (v: T) => void
  let reject!: (e: unknown) => void
  const promise = new Promise<T>((res, rej) => { resolve = res; reject = rej })
  return { promise, resolve, reject }
}

const ids = (store: ReturnType<typeof useBoardStore>, status: string) =>
  store.columns.find(c => c.status === status)!.issues.map(i => i.id)

describe('useBoardStore moveIssue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.resetAllMocks()
  })

  async function loaded(content: Record<string, unknown[]>) {
    board.getBoard.mockResolvedValue(boardOf(content))
    const store = useBoardStore()
    await store.load('P')
    return store
  }

  it('moves the card before the save has finished', async () => {
    const save = deferred<never>()
    issues.update.mockReturnValue(save.promise)
    const store = await loaded({ todo: [issue(1, 'todo')] })

    const moving = store.moveIssue('P', 1, 'in_progress')
    expect(ids(store, 'todo')).toEqual([])
    expect(ids(store, 'in_progress')).toEqual([1])
    expect(store.columns[1]!.issues[0]!.status).toBe('in_progress')

    save.resolve({} as never)
    await moving
  })

  it('places the card by priority then id, not at the end', async () => {
    issues.update.mockResolvedValue({} as never)
    const store = await loaded({
      todo: [issue(5, 'todo', 'high')],
      in_progress: [issue(2, 'in_progress', 'critical'), issue(9, 'in_progress', 'low')],
    })
    await store.moveIssue('P', 5, 'in_progress')
    expect(ids(store, 'in_progress')).toEqual([2, 5, 9])
  })

  it('sends only the new status', async () => {
    issues.update.mockResolvedValue({} as never)
    const store = await loaded({ todo: [issue(1, 'todo')] })
    await store.moveIssue('P', 1, 'done')
    expect(issues.update).toHaveBeenCalledTimes(1)
    expect(issues.update).toHaveBeenCalledWith('P', 1, { status: 'done' })
  })

  it('changes the count of both columns', async () => {
    issues.update.mockResolvedValue({} as never)
    const store = await loaded({ todo: [issue(1, 'todo'), issue(2, 'todo')], in_review: [issue(3, 'in_review')] })
    await store.moveIssue('P', 1, 'in_review')
    expect(store.columns.map(c => c.issues.length)).toEqual([1, 0, 2, 0])
  })

  it('does nothing when the card is dropped on its own column', async () => {
    const store = await loaded({ todo: [issue(1, 'todo')] })
    await store.moveIssue('P', 1, 'todo')
    expect(issues.update).not.toHaveBeenCalled()
    expect(ids(store, 'todo')).toEqual([1])
    expect(store.announcement).toBe('')
  })

  it('ignores a card that is not on the board', async () => {
    const store = await loaded({ todo: [issue(1, 'todo')] })
    await store.moveIssue('P', 99, 'done')
    expect(issues.update).not.toHaveBeenCalled()
  })

  it('announces the move once it is saved', async () => {
    issues.update.mockResolvedValue({} as never)
    const store = await loaded({ todo: [issue(1, 'todo')] })
    await store.moveIssue('P', 1, 'in_progress')
    expect(store.announcement).toBe('P-1 moved to In Progress')
  })

  it('sends two quick moves of the same card in the order they were made', async () => {
    const first = deferred<never>()
    const second = deferred<never>()
    issues.update.mockReturnValueOnce(first.promise).mockReturnValueOnce(second.promise)
    const store = await loaded({ todo: [issue(1, 'todo')] })

    const a = store.moveIssue('P', 1, 'in_progress')
    const b = store.moveIssue('P', 1, 'done')
    // The board already shows the latest move, but only the first save is out.
    expect(ids(store, 'done')).toEqual([1])
    expect(issues.update).toHaveBeenCalledTimes(1)
    expect(issues.update).toHaveBeenLastCalledWith('P', 1, { status: 'in_progress' })

    first.resolve({} as never)
    await a
    await new Promise(r => setTimeout(r))
    expect(issues.update).toHaveBeenCalledTimes(2)
    expect(issues.update).toHaveBeenLastCalledWith('P', 1, { status: 'done' })

    second.resolve({} as never)
    await b
  })
})

describe('useBoardStore failed moves', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.resetAllMocks()
  })

  async function loaded(content: Record<string, unknown[]>) {
    board.getBoard.mockResolvedValue(boardOf(content))
    const store = useBoardStore()
    await store.load('P')
    return store
  }

  it('puts the card back, in the place the ordering gives it, and reports the failure', async () => {
    issues.update.mockRejectedValue(new Error('network down'))
    const store = await loaded({
      todo: [issue(1, 'todo', 'high'), issue(2, 'todo', 'low')],
    })
    await store.moveIssue('P', 1, 'in_progress')
    expect(ids(store, 'todo')).toEqual([1, 2])
    expect(ids(store, 'in_progress')).toEqual([])
    expect(store.error).toBeNull()
    expect(store.moveFailure?.message).toBe('P-1 could not be moved')
    expect(store.announcement).toBe('P-1 could not be moved')
  })

  it('shows the reason the server gave', async () => {
    issues.update.mockRejectedValue({ data: { message: 'forbidden' } })
    const store = await loaded({ todo: [issue(1, 'todo')] })
    await store.moveIssue('P', 1, 'done')
    expect(store.moveFailure?.message).toBe('P-1 could not be moved: forbidden')
  })

  it('a second failure is a new failure, so the page can show it again', async () => {
    issues.update.mockRejectedValue(new Error('x'))
    const store = await loaded({ todo: [issue(1, 'todo')] })
    await store.moveIssue('P', 1, 'done')
    const first = store.moveFailure?.seq
    await store.moveIssue('P', 1, 'done')
    expect(store.moveFailure?.seq).not.toBe(first)
  })

  it('leaves other cards alone when one move fails', async () => {
    issues.update.mockRejectedValue(new Error('x'))
    const store = await loaded({ todo: [issue(1, 'todo'), issue(2, 'todo')], done: [issue(3, 'done')] })
    await store.moveIssue('P', 2, 'in_review')
    expect(ids(store, 'todo')).toEqual([1, 2])
    expect(ids(store, 'done')).toEqual([3])
  })

  it('reloads the board when the task no longer exists', async () => {
    issues.update.mockRejectedValue({ statusCode: 404, data: { message: 'not found' } })
    const store = await loaded({ todo: [issue(1, 'todo'), issue(2, 'todo')] })
    board.getBoard.mockResolvedValue(boardOf({ todo: [issue(2, 'todo')] }))
    await store.moveIssue('P', 1, 'done')
    expect(board.getBoard).toHaveBeenCalledTimes(2)
    expect(ids(store, 'todo')).toEqual([2])
    expect(ids(store, 'done')).toEqual([])
    expect(store.moveFailure).not.toBeNull()
  })

  it('does not move the card back when an earlier move fails but a later move is still pending', async () => {
    const first = deferred<never>()
    const second = deferred<never>()
    issues.update.mockReturnValueOnce(first.promise).mockReturnValueOnce(second.promise)
    const store = await loaded({ todo: [issue(1, 'todo')] })

    const a = store.moveIssue('P', 1, 'in_progress')
    const b = store.moveIssue('P', 1, 'done')
    first.reject(new Error('x'))
    await a
    expect(ids(store, 'done')).toEqual([1])
    expect(store.moveFailure).toBeNull()

    await new Promise(r => setTimeout(r))
    second.resolve({} as never)
    await b
    expect(ids(store, 'done')).toEqual([1])
    expect(store.moveFailure).toBeNull()
    expect(store.announcement).toBe('P-1 moved to Done')
  })

  it('puts the card back to the status the server really holds when the later move fails too', async () => {
    const first = deferred<never>()
    const second = deferred<never>()
    issues.update.mockReturnValueOnce(first.promise).mockReturnValueOnce(second.promise)
    const store = await loaded({ todo: [issue(1, 'todo')] })

    const a = store.moveIssue('P', 1, 'in_progress')
    const b = store.moveIssue('P', 1, 'done')
    first.reject(new Error('x'))
    await a
    await new Promise(r => setTimeout(r))
    second.reject(new Error('y'))
    await b
    // Not the intermediate column: the first move never reached the server either.
    expect(ids(store, 'todo')).toEqual([1])
    expect(ids(store, 'in_progress')).toEqual([])
    expect(store.moveFailure?.message).toBe('P-1 could not be moved')
  })

  it('puts the card back to the last saved status when a later move fails after an earlier one was saved', async () => {
    const first = deferred<never>()
    const second = deferred<never>()
    issues.update.mockReturnValueOnce(first.promise).mockReturnValueOnce(second.promise)
    const store = await loaded({ todo: [issue(1, 'todo')] })

    const a = store.moveIssue('P', 1, 'in_progress')
    const b = store.moveIssue('P', 1, 'done')
    first.resolve({} as never)
    await a
    await new Promise(r => setTimeout(r))
    second.reject(new Error('y'))
    await b
    expect(ids(store, 'in_progress')).toEqual([1])
    expect(ids(store, 'done')).toEqual([])
  })

  it('never rejects, so a caller that does not await it cannot raise an unhandled error', async () => {
    issues.update.mockRejectedValue(new Error('x'))
    const store = await loaded({ todo: [issue(1, 'todo')] })
    await expect(store.moveIssue('P', 1, 'done')).resolves.toBeUndefined()
  })
})

describe('useBoardStore refreshIfStale', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.resetAllMocks()
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-10-29T10:00:00Z'))
  })
  afterEach(() => vi.useRealTimers())

  async function loaded(content: Record<string, unknown[]> = { todo: [issue(1, 'todo')] }) {
    board.getBoard.mockResolvedValue(boardOf(content))
    const store = useBoardStore()
    await store.load('P')
    return store
  }

  it('does nothing when the board was loaded less than 30 seconds ago', async () => {
    const store = await loaded()
    await vi.advanceTimersByTimeAsync(29_000)
    await store.refreshIfStale('P')
    expect(board.getBoard).toHaveBeenCalledTimes(1)
  })

  it('refetches quietly once the board is 30 seconds old: no loading state, columns stay put meanwhile', async () => {
    const store = await loaded()
    await vi.advanceTimersByTimeAsync(30_000)
    const fresh = deferred<never>()
    board.getBoard.mockReturnValue(fresh.promise)

    const refreshing = store.refreshIfStale('P')
    expect(store.loading).toBe(false)
    expect(ids(store, 'todo')).toEqual([1])

    fresh.resolve(boardOf({ done: [issue(2, 'done')] }) as never)
    await refreshing
    expect(store.loading).toBe(false)
    expect(ids(store, 'todo')).toEqual([])
    expect(ids(store, 'done')).toEqual([2])
  })

  it('counts the refresh as a load, so an immediate second refresh does nothing', async () => {
    const store = await loaded()
    await vi.advanceTimersByTimeAsync(31_000)
    await store.refreshIfStale('P')
    await store.refreshIfStale('P')
    expect(board.getBoard).toHaveBeenCalledTimes(2)
  })

  it('does nothing while a move is still being saved', async () => {
    const save = deferred<never>()
    issues.update.mockReturnValue(save.promise)
    const store = await loaded()
    const moving = store.moveIssue('P', 1, 'done')
    await vi.advanceTimersByTimeAsync(60_000)

    await store.refreshIfStale('P')
    expect(board.getBoard).toHaveBeenCalledTimes(1)

    save.resolve({} as never)
    await moving
  })

  it('throws the answer away if a move started after the request began', async () => {
    const store = await loaded()
    await vi.advanceTimersByTimeAsync(31_000)
    const stale = deferred<never>()
    board.getBoard.mockReturnValue(stale.promise)
    issues.update.mockResolvedValue({} as never)

    const refreshing = store.refreshIfStale('P')
    await store.moveIssue('P', 1, 'done')
    stale.resolve(boardOf({ todo: [issue(1, 'todo')] }) as never) // the old picture, from before the move
    await refreshing

    expect(ids(store, 'done')).toEqual([1])
    expect(ids(store, 'todo')).toEqual([])
  })

  it('keeps the current columns and reports nothing when the refresh fails', async () => {
    const store = await loaded()
    await vi.advanceTimersByTimeAsync(31_000)
    board.getBoard.mockRejectedValue(new Error('offline'))

    await expect(store.refreshIfStale('P')).resolves.toBeUndefined()
    expect(ids(store, 'todo')).toEqual([1])
    expect(store.error).toBeNull()
    expect(store.moveFailure).toBeNull()
  })
})

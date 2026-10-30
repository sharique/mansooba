import { defineStore } from 'pinia'
import { boardService } from '~/services/board.service'
import { issuesService } from '~/services/issues.service'
import type { BoardColumn } from '~/services/board.service'
import { sortBoardIssues } from '~/utils/boardOrder'
import { statusLabel } from '~/utils/issueStyles'
import type { Issue } from '~/types/domain.types'

// How long a loaded board is trusted before coming back to it refreshes it.
const REFRESH_AFTER_MS = 30_000

function messageOf(e: unknown): string {
  const err = e as { data?: { message?: string }; message?: string }
  return err?.data?.message ?? err?.message ?? 'Something went wrong'
}

export const useBoardStore = defineStore('board', () => {
  const columns = ref<BoardColumn[]>([])
  // Starts true so the first paint is a skeleton, not an empty board.
  const loading = ref(true)
  const error = ref<string | null>(null)
  // Text for the page's live region: the result of the last move, spoken to screen readers.
  const announcement = ref('')
  // The last move that could not be saved. `seq` changes on every failure so the page can
  // show the same message twice.
  const moveFailure = ref<{ seq: number; message: string } | null>(null)
  let failureSeq = 0
  let lastLoadedAt = 0
  // Counts moves, so a refresh that started before a move can tell its answer is out of date.
  let moveCount = 0

  // Columns are sorted here as well as on the server, so the page never depends on
  // the order the API happened to return.
  function applyBoard(data: { columns: BoardColumn[] }) {
    columns.value = data.columns.map(c => ({ ...c, issues: sortBoardIssues(c.issues) }))
  }

  async function load(projectKey: string) {
    loading.value = true
    error.value = null
    try {
      applyBoard(await boardService.getBoard(projectKey))
      lastLoadedAt = Date.now()
    }
    catch (e) {
      columns.value = []
      error.value = messageOf(e)
    }
    finally {
      loading.value = false
    }
  }

  // The newest save for each card, so a second move of the same card waits for the first
  // and the server sees the moves in the order they were made.
  const saves = new Map<number, Promise<unknown>>()
  // The status the server holds for a card whose moves are still being saved: where the
  // card goes back to if the last of those moves fails.
  const confirmed = new Map<number, Issue['status']>()

  function findCard(issueId: number): Issue | undefined {
    for (const c of columns.value) {
      const hit = c.issues.find(i => i.id === issueId)
      if (hit) return hit
    }
    return undefined
  }

  // Takes the card out of its column and puts it into `toStatus`, in the place the
  // board ordering gives it (not at the end).
  function place(card: Issue, toStatus: Issue['status']) {
    const moved = { ...card, status: toStatus }
    columns.value = columns.value.map((c) => {
      const without = c.issues.filter(i => i.id !== card.id)
      return { ...c, issues: c.status === toStatus ? sortBoardIssues([...without, moved]) : without }
    })
  }

  // Fetches the board without the loading state, for when the board must be corrected
  // while the person is looking at it.
  async function reloadQuietly(projectKey: string) {
    try {
      applyBoard(await boardService.getBoard(projectKey))
      lastLoadedAt = Date.now()
    }
    catch {
      // Keep what is on screen; the next load will try again.
    }
  }

  async function settleFailure(projectKey: string, card: Issue, e: unknown) {
    const gone = [(e as { statusCode?: number }).statusCode, (e as { status?: number }).status].includes(404)
    if (gone) {
      await reloadQuietly(projectKey)
    }
    else {
      const current = findCard(card.id)
      const real = confirmed.get(card.id)
      if (current && real) place(current, real)
    }
    const detail = (e as { data?: { message?: string } })?.data?.message
    const message = detail ? `${card.key} could not be moved: ${detail}` : `${card.key} could not be moved`
    moveFailure.value = { seq: ++failureSeq, message }
    announcement.value = message
  }

  // Moves a card now and saves it in the background. It never throws: a failure puts the
  // card back and is reported through `moveFailure`.
  async function moveIssue(projectKey: string, issueId: number, toStatus: Issue['status']) {
    const card = findCard(issueId)
    if (!card || card.status === toStatus) return

    moveCount++
    if (!saves.has(issueId)) confirmed.set(issueId, card.status)
    place(card, toStatus)

    const send = () => issuesService.update(projectKey, issueId, { status: toStatus })
    const earlier = saves.get(issueId)
    const run = earlier ? earlier.then(send, send) : send()
    saves.set(issueId, run)
    try {
      await run
      confirmed.set(issueId, toStatus)
      if (saves.get(issueId) === run) announcement.value = `${card.key} moved to ${statusLabel(toStatus)}`
    }
    catch (e) {
      // A failed move followed by another move of the same card is settled by that later
      // move; only the last move of a card puts it back.
      if (saves.get(issueId) === run) await settleFailure(projectKey, card, e)
    }
    finally {
      if (saves.get(issueId) === run) {
        saves.delete(issueId)
        confirmed.delete(issueId)
      }
    }
  }

  // Called when the person comes back to the board. Quietly fetches a fresh copy if the
  // current one is old enough, unless a save is in flight or a move happens meanwhile.
  async function refreshIfStale(projectKey: string) {
    if (loading.value || saves.size > 0) return
    if (Date.now() - lastLoadedAt < REFRESH_AFTER_MS) return
    const movesSeen = moveCount
    try {
      const data = await boardService.getBoard(projectKey)
      if (moveCount !== movesSeen || saves.size > 0) return
      applyBoard(data)
      lastLoadedAt = Date.now()
    }
    catch {
      // Quiet by design: keep the columns on screen and say nothing.
    }
  }

  return { columns, loading, error, announcement, moveFailure, load, moveIssue, refreshIfStale }
})

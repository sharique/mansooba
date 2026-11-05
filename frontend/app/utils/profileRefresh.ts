interface Deps {
  isAuthenticated: () => boolean
  fetchMe: () => Promise<unknown>
}

/**
 * The profile (with the system administrator flag) is saved in the browser, so a
 * change made by someone else would stay hidden until the next sign-in. Reload it
 * on start and whenever the tab becomes visible again. The server decides access
 * on every request anyway; this only keeps the menus honest.
 * `minGapMs` limits how often it asks. Returns a function that stops it.
 */
export function startProfileRefresh(deps: Deps, minGapMs = 30_000): () => void {
  let last = Number.NEGATIVE_INFINITY
  const refresh = () => {
    if (!deps.isAuthenticated()) return
    const now = Date.now()
    if (now - last < minGapMs) return
    last = now
    deps.fetchMe().catch(() => { /* keep what is shown */ })
  }
  const onVisible = () => {
    if (document.visibilityState === 'visible') refresh()
  }
  refresh()
  document.addEventListener('visibilitychange', onVisible)
  return () => document.removeEventListener('visibilitychange', onVisible)
}

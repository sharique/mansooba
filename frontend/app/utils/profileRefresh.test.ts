// @vitest-environment happy-dom
import { describe, expect, it, vi } from 'vitest'
import { startProfileRefresh } from './profileRefresh'

const setup = (authenticated = true) => {
  const fetchMe = vi.fn().mockResolvedValue(undefined)
  const stop = startProfileRefresh({ isAuthenticated: () => authenticated, fetchMe }, 0)
  return { fetchMe, stop }
}

describe('profile refresh', () => {
  it('reloads the profile on start so a changed system role shows after a page reload', () => {
    expect(setup().fetchMe).toHaveBeenCalledTimes(1)
  })

  it('reloads again when the tab becomes visible', () => {
    const { fetchMe } = setup()
    document.dispatchEvent(new Event('visibilitychange'))
    expect(fetchMe).toHaveBeenCalledTimes(2)
  })

  it('does nothing while signed out', () => {
    const { fetchMe } = setup(false)
    document.dispatchEvent(new Event('visibilitychange'))
    expect(fetchMe).not.toHaveBeenCalled()
  })

  it('swallows failures and stops listening when asked', () => {
    const { fetchMe, stop } = setup()
    fetchMe.mockRejectedValue(new Error('x'))
    stop()
    document.dispatchEvent(new Event('visibilitychange'))
    expect(fetchMe).toHaveBeenCalledTimes(1)
  })
})

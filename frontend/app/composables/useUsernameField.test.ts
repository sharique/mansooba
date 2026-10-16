import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { useUsernameField } from './useUsernameField'

const flush = async () => { await nextTick(); await Promise.resolve(); await nextTick() }

describe('useUsernameField', () => {
  beforeEach(() => { vi.useFakeTimers() })
  afterEach(() => { vi.useRealTimers() })

  it('follows the email while the person has not edited it', async () => {
    const email = ref('')
    const f = useUsernameField(email, { check: null })
    expect(f.username.value).toBe('')
    email.value = 'Sam-Smith@example.com'
    await flush()
    expect(f.username.value).toBe('samsmith')
    email.value = 'admin@example.com'
    await flush()
    expect(f.username.value).toBe('admin2')
  })

  it('keeps an edited username when the email changes afterwards', async () => {
    const email = ref('sam@example.com')
    const f = useUsernameField(email, { check: null })
    await flush()
    f.onInput('my_name')
    email.value = 'someone.else@example.com'
    await flush()
    expect(f.username.value).toBe('my_name')
    expect(f.touched.value).toBe(true)
  })

  it('reports rule errors as the person types, without asking the server', async () => {
    const check = vi.fn()
    const email = ref('')
    const f = useUsernameField(email, { check })
    f.onInput('ab')
    expect(f.status.value).toMatchObject({ kind: 'invalid' })
    f.onInput('sam-smith')
    expect(f.status.value).toMatchObject({ kind: 'invalid' })
    f.onInput('admin')
    expect(f.status.value).toMatchObject({ kind: 'reserved' })
    await vi.advanceTimersByTimeAsync(1000)
    expect(check).not.toHaveBeenCalled()
  })

  it('asks the server about a valid name about 400 ms after the last keystroke', async () => {
    const check = vi.fn().mockResolvedValue({ available: true, reason: '', message: '' })
    const f = useUsernameField(ref(''), { check })
    f.onInput('fresh_name')
    expect(f.status.value.kind).toBe('checking')
    await vi.advanceTimersByTimeAsync(399)
    expect(check).not.toHaveBeenCalled()
    f.onInput('fresh_name2') // typing again restarts the wait
    await vi.advanceTimersByTimeAsync(399)
    expect(check).not.toHaveBeenCalled()
    await vi.advanceTimersByTimeAsync(2)
    expect(check).toHaveBeenCalledTimes(1)
    expect(check).toHaveBeenCalledWith('fresh_name2')
    await flush()
    expect(f.status.value.kind).toBe('available')
  })

  it('shows taken from the server verdict', async () => {
    const check = vi.fn().mockResolvedValue({ available: false, reason: 'taken', message: 'That username is already taken.' })
    const f = useUsernameField(ref(''), { check })
    f.onInput('sam')
    await vi.advanceTimersByTimeAsync(500)
    await flush()
    expect(f.status.value).toMatchObject({ kind: 'taken', message: 'That username is already taken.' })
  })

  it('shows no verdict when the check fails, and never blocks typing', async () => {
    const check = vi.fn().mockRejectedValue(new Error('network'))
    const f = useUsernameField(ref(''), { check })
    f.onInput('fresh_name')
    await vi.advanceTimersByTimeAsync(500)
    await flush()
    expect(f.status.value.kind).toBe('unknown')
    f.onInput('another_one')
    expect(f.username.value).toBe('another_one')
  })

  it('does not ask the server when the check is disabled (first administrator step)', async () => {
    const f = useUsernameField(ref(''), { check: null })
    f.onInput('fresh_name')
    await vi.advanceTimersByTimeAsync(1000)
    expect(f.status.value.kind).toBe('unknown')
  })

  it('ignores a slow answer for a name that is no longer in the field', async () => {
    let resolveFirst: (v: unknown) => void = () => {}
    const check = vi.fn()
      .mockImplementationOnce(() => new Promise((r) => { resolveFirst = r }))
      .mockResolvedValueOnce({ available: true, reason: '', message: '' })
    const f = useUsernameField(ref(''), { check })
    f.onInput('first_name')
    await vi.advanceTimersByTimeAsync(500)
    f.onInput('second_name')
    await vi.advanceTimersByTimeAsync(500)
    await flush()
    expect(f.status.value.kind).toBe('available')
    resolveFirst({ available: false, reason: 'taken', message: 'That username is already taken.' })
    await flush()
    expect(f.status.value.kind).toBe('available')
  })

  it('exposes a server error from submit on the field', () => {
    const f = useUsernameField(ref(''), { check: null })
    f.onInput('sam_ok')
    f.setServerError('That username is already taken.')
    expect(f.status.value).toMatchObject({ kind: 'taken', message: 'That username is already taken.' })
  })
})

import { beforeEach, describe, expect, test, vi } from 'vitest'

const mockApi = vi.fn()
vi.stubGlobal('useNuxtApp', () => ({ $api: mockApi }))

describe('useSystemLogs', () => {
  beforeEach(() => mockApi.mockReset())

  test('fetchLogs calls GET /admin/system-logs with default page/size, no filters', async () => {
    const { useSystemLogs } = await import('./useSystemLogs')
    mockApi.mockResolvedValueOnce({ entries: [], total: 0, page: 1, size: 20 })
    const { fetchLogs } = useSystemLogs()
    await fetchLogs()
    expect(mockApi).toHaveBeenCalledWith('/admin/system-logs?page=1&size=20')
  })

  test('fetchLogs includes category, from, to, actor, and q when provided', async () => {
    const { useSystemLogs } = await import('./useSystemLogs')
    mockApi.mockResolvedValueOnce({ entries: [], total: 0, page: 1, size: 20 })
    const { fetchLogs } = useSystemLogs()
    await fetchLogs(
      {
        category: 'authentication',
        from: '2026-07-01T00:00:00Z',
        to: '2026-07-31T23:59:59Z',
        actor: 'jane@example.com',
        q: 'failed',
      },
      1,
      20,
    )
    const calledWith = mockApi.mock.calls[0][0] as string
    expect(calledWith).toContain('category=authentication')
    expect(calledWith).toContain('from=2026-07-01T00%3A00%3A00Z')
    expect(calledWith).toContain('to=2026-07-31T23%3A59%3A59Z')
    expect(calledWith).toContain('actor=jane%40example.com')
    expect(calledWith).toContain('q=failed')
  })

  test('fetchLogs uses the given page and size', async () => {
    const { useSystemLogs } = await import('./useSystemLogs')
    mockApi.mockResolvedValueOnce({ entries: [], total: 0, page: 2, size: 50 })
    const { fetchLogs } = useSystemLogs()
    await fetchLogs({}, 2, 50)
    expect(mockApi).toHaveBeenCalledWith('/admin/system-logs?page=2&size=50')
  })

  test('populates entries, total, and page from the response', async () => {
    const { useSystemLogs } = await import('./useSystemLogs')
    mockApi.mockResolvedValueOnce({
      entries: [
        { id: '1-abc', event_category: 'authentication', action: 'login_failed', outcome: 'failure', actor_label: 'jane@example.com', target_label: null, detail: '', created_at: '2026-07-26T14:00:00Z' },
      ],
      total: 1,
      page: 1,
      size: 20,
    })
    const { fetchLogs, entries, total, page } = useSystemLogs()
    await fetchLogs()
    expect(entries.value).toHaveLength(1)
    expect(entries.value[0].actor_label).toBe('jane@example.com')
    expect(total.value).toBe(1)
    expect(page.value).toBe(1)
  })

  test('sets error on failure and clears loading', async () => {
    const { useSystemLogs } = await import('./useSystemLogs')
    mockApi.mockRejectedValueOnce(new Error('network error'))
    const { fetchLogs, error, loading } = useSystemLogs()
    await fetchLogs()
    expect(error.value).toBe('network error')
    expect(loading.value).toBe(false)
  })
})

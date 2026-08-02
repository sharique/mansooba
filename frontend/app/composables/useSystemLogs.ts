import type { SystemLogEntry, SystemLogListFilters, SystemLogListResponse } from '~/types/domain.types'

export function useSystemLogs() {
  const { $api } = useNuxtApp()

  const entries = ref<SystemLogEntry[]>([])
  const total = ref(0)
  const page = ref(1)
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchLogs(filters: SystemLogListFilters = {}, p = 1, size = 20) {
    loading.value = true
    error.value = null
    try {
      const params = new URLSearchParams()
      if (filters.category) params.set('category', filters.category)
      if (filters.from) params.set('from', filters.from)
      if (filters.to) params.set('to', filters.to)
      if (filters.actor) params.set('actor', filters.actor)
      if (filters.q) params.set('q', filters.q)
      params.set('page', String(p))
      params.set('size', String(size))

      const data = await $api<SystemLogListResponse>(`/admin/system-logs?${params.toString()}`)
      entries.value = data.entries
      total.value = data.total
      page.value = data.page
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to load system logs'
    } finally {
      loading.value = false
    }
  }

  return { entries, total, page, loading, error, fetchLogs }
}

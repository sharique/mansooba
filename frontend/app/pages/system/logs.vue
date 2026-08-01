<template>
  <div class="max-w-6xl mx-auto py-8 px-4">
    <h1 class="text-2xl font-bold mb-6">System Logs</h1>

    <!-- Filter/search controls -->
    <div class="flex flex-wrap gap-3 mb-6 items-end">
      <div class="form-control">
        <label class="label"><span class="label-text">Category</span></label>
        <select v-model="filters.category" class="select select-bordered select-sm" @change="applyFilters">
          <option value="">All categories</option>
          <option value="authentication">Authentication</option>
          <option value="admin_action">Admin action</option>
          <option value="settings_change">Settings change</option>
          <option value="db_lifecycle">Database lifecycle</option>
        </select>
      </div>

      <div class="form-control">
        <label class="label"><span class="label-text">From</span></label>
        <input v-model="filters.from" type="date" class="input input-bordered input-sm" @change="applyFilters" />
      </div>

      <div class="form-control">
        <label class="label"><span class="label-text">To</span></label>
        <input v-model="filters.to" type="date" class="input input-bordered input-sm" @change="applyFilters" />
      </div>

      <div class="form-control">
        <label class="label"><span class="label-text">Actor</span></label>
        <input
          v-model="filters.actor"
          type="text"
          placeholder="email"
          class="input input-bordered input-sm"
          @keyup.enter="applyFilters"
        />
      </div>

      <div class="form-control flex-1 min-w-48">
        <label class="label"><span class="label-text">Search</span></label>
        <input
          v-model="filters.q"
          type="text"
          placeholder="Free-text search"
          class="input input-bordered input-sm w-full"
          @keyup.enter="applyFilters"
        />
      </div>

      <button class="btn btn-sm btn-primary" @click="applyFilters">Apply</button>
      <button class="btn btn-sm btn-ghost" @click="clearFilters">Clear</button>
    </div>

    <!-- Error banner -->
    <div v-if="error" class="alert alert-error mb-4 flex items-center gap-2">
      <span>{{ error }}</span>
      <button class="btn btn-sm" @click="applyFilters">Retry</button>
    </div>

    <!-- Loading skeleton -->
    <div v-if="loading" class="space-y-2">
      <div v-for="n in 5" :key="n" class="skeleton h-10 w-full rounded" />
    </div>

    <!-- Entries table -->
    <div v-else-if="entries.length" class="overflow-x-auto">
      <table class="table table-zebra w-full">
        <thead>
          <tr>
            <th>Timestamp</th>
            <th>Category</th>
            <th>Action</th>
            <th>Actor</th>
            <th>Target</th>
            <th>Outcome</th>
            <th>Detail</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="entry in entries" :key="entry.id">
            <td class="whitespace-nowrap">{{ formatTimestamp(entry.created_at) }}</td>
            <td>
              <span class="badge badge-ghost">{{ entry.event_category }}</span>
            </td>
            <td>{{ entry.action }}</td>
            <td>{{ entry.actor_label }}</td>
            <td>{{ entry.target_label ?? '—' }}</td>
            <td>
              <span :class="entry.outcome === 'success' ? 'badge badge-success' : 'badge badge-error'">
                {{ entry.outcome }}
              </span>
            </td>
            <td class="max-w-xs truncate" :title="entry.detail">{{ entry.detail }}</td>
          </tr>
        </tbody>
      </table>

      <!-- Pagination -->
      <div v-if="total > pageSize" class="flex justify-center gap-2 mt-4">
        <button class="btn btn-sm" :disabled="page <= 1" @click="goToPage(page - 1)">«</button>
        <span class="flex items-center px-2">Page {{ page }} / {{ totalPages }}</span>
        <button class="btn btn-sm" :disabled="page >= totalPages" @click="goToPage(page + 1)">»</button>
      </div>
    </div>

    <!-- Empty state -->
    <div v-else class="text-center py-16 text-base-content/50">
      No log entries match the current filters.
    </div>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '~/stores/auth.store'
import { useSystemLogs } from '~/composables/useSystemLogs'
import type { SystemLogListFilters } from '~/types/domain.types'

const authStore = useAuthStore()
const { entries, total, page, loading, error, fetchLogs } = useSystemLogs()

const pageSize = 20
const totalPages = computed(() => Math.ceil(total.value / pageSize))

const filters = reactive<SystemLogListFilters>({
  category: '',
  from: '',
  to: '',
  actor: '',
  q: '',
})

onMounted(async () => {
  if (!authStore.isAdmin) {
    await navigateTo('/')
    return
  }
  await fetchLogs(activeFilters(), 1, pageSize)
})

// Converts a plain <input type="date"> value ("2026-07-01") into the
// RFC3339 bound the API expects (FR-006), and strips empty filter fields
// entirely rather than sending them as empty strings.
function activeFilters(): SystemLogListFilters {
  const active: SystemLogListFilters = {}
  if (filters.category) active.category = filters.category
  if (filters.from) active.from = `${filters.from}T00:00:00Z`
  if (filters.to) active.to = `${filters.to}T23:59:59Z`
  if (filters.actor) active.actor = filters.actor
  if (filters.q) active.q = filters.q
  return active
}

async function applyFilters() {
  await fetchLogs(activeFilters(), 1, pageSize)
}

async function clearFilters() {
  filters.category = ''
  filters.from = ''
  filters.to = ''
  filters.actor = ''
  filters.q = ''
  await applyFilters()
}

async function goToPage(p: number) {
  await fetchLogs(activeFilters(), p, pageSize)
}

function formatTimestamp(iso: string): string {
  return new Date(iso).toLocaleString()
}
</script>

<template>
  <section class="card bg-base-100 border border-base-300" aria-labelledby="overview-sprint-title">
    <div class="card-body gap-3">
      <div class="flex items-center justify-between gap-2">
        <h2 id="overview-sprint-title" class="card-title text-base">Current sprint</h2>
        <NuxtLink :to="`/projects/${projectKey}/sprints`" class="link link-hover text-sm">Sprints</NuxtLink>
      </div>

      <div v-if="loading" class="space-y-2">
        <div class="skeleton h-5 w-48" />
        <div class="skeleton h-4 w-64" />
        <div class="skeleton h-16 w-full" />
      </div>

      <div v-else-if="error" class="alert alert-error text-sm" role="alert">
        <span class="flex-1">Could not load the sprint: {{ error }}</span>
        <button class="btn btn-sm" @click="emit('retry')">Retry</button>
      </div>

      <p v-else-if="!data" class="text-sm text-base-content/60">
        No active sprint. Start one from the
        <NuxtLink :to="`/projects/${projectKey}/sprints`" class="link">Sprints page</NuxtLink>.
      </p>

      <template v-else>
        <div class="flex flex-wrap items-baseline gap-x-3 gap-y-1">
          <span class="font-semibold">{{ data.name }}</span>
          <span class="text-sm text-base-content/60">{{ formatDate(data.start_date) }} – {{ formatDate(data.end_date) }}</span>
          <span v-if="data.is_overdue" class="badge badge-warning badge-sm">Overdue</span>
        </div>
        <p v-if="data.total.tasks === 0" class="text-sm text-base-content/60">This sprint has no tasks yet.</p>
        <dl class="grid grid-cols-2 gap-2 sm:grid-cols-4">
          <div
            v-for="c in counters"
            :key="c.key"
            :data-counter="c.key"
            class="rounded-box border border-base-300 p-3"
          >
            <dt class="text-xs uppercase tracking-wide text-base-content/60">{{ c.label }}</dt>
            <dd class="text-2xl font-bold">{{ c.count.tasks }}</dd>
            <dd class="text-xs text-base-content/60">{{ c.count.points }} pts</dd>
          </div>
        </dl>
      </template>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { SprintSummary, StatusCount } from '~/types/domain.types'

const props = defineProps<{
  projectKey: string
  data: SprintSummary | null
  loading: boolean
  error: string | null
}>()
const emit = defineEmits<{ (e: 'retry'): void }>()

const { formatDate } = useTimeFormatter()

// The same four columns, in the same order, as the board.
const counters = computed<{ key: string; label: string; count: StatusCount }[]>(() => {
  const d = props.data
  if (!d) return []
  return [
    { key: 'todo', label: 'To Do', count: d.todo },
    { key: 'in_progress', label: 'In Progress', count: d.in_progress },
    { key: 'in_review', label: 'In Review', count: d.in_review },
    { key: 'done', label: 'Done', count: d.done },
  ]
})
</script>

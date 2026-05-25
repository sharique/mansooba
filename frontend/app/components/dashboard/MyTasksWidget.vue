<template>
  <div class="card bg-base-100 shadow border border-base-200">
    <div class="card-body">
      <h2 class="card-title text-base mb-2">My Tasks</h2>

      <!-- Skeleton -->
      <div v-if="loading" class="space-y-2">
        <div v-for="i in 5" :key="i" class="skeleton h-10 w-full rounded" />
      </div>

      <!-- Empty state -->
      <p v-else-if="sorted.length === 0" class="text-base-content/40 text-sm py-4">
        No tasks assigned to you yet.
      </p>

      <!-- Task list -->
      <div v-else class="divide-y divide-base-200">
        <NuxtLink
          v-for="task in sorted"
          :key="task.id"
          :to="`/projects/${projectKey(task.key)}/issues/${task.id}`"
          class="flex items-center gap-3 py-2 hover:bg-base-200 -mx-2 px-2 rounded transition-colors"
        >
          <!-- Priority dot -->
          <span :class="['w-2 h-2 rounded-full shrink-0', priorityColor(task.priority)]" />

          <!-- Issue key + title -->
          <span class="text-xs text-base-content/40 w-24 shrink-0 font-mono">{{ task.key }}</span>
          <span class="flex-1 text-sm truncate">{{ task.title }}</span>

          <!-- Status badge -->
          <span :class="['badge badge-sm shrink-0', statusBadge(task.status)]">
            {{ statusLabel(task.status) }}
          </span>
        </NuxtLink>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '~/stores/auth.store'
import type { Issue } from '~/types/domain.types'

defineProps<{ loading: boolean }>()

const authStore = useAuthStore()

const STATUS_ORDER: Record<string, number> = {
  in_progress: 0,
  todo:        1,
  in_review:   2,
  backlog:     3,
  done:        4,
}

const sorted = computed(() =>
  [...authStore.myIssues]
    .sort((a, b) => (STATUS_ORDER[a.status] ?? 9) - (STATUS_ORDER[b.status] ?? 9))
    .slice(0, 10),
)

// Extract project key from issue key (e.g. "myproj-42" → "myproj")
function projectKey(issueKey: string): string {
  return issueKey.replace(/-\d+$/, '')
}

function priorityColor(priority: Issue['priority']): string {
  switch (priority) {
    case 'critical': return 'bg-error'
    case 'high':     return 'bg-warning'
    case 'medium':   return 'bg-info'
    default:         return 'bg-base-300'
  }
}

function statusBadge(status: Issue['status']): string {
  switch (status) {
    case 'in_progress': return 'badge-primary'
    case 'in_review':   return 'badge-secondary'
    case 'done':        return 'badge-success'
    default:            return 'badge-ghost'
  }
}

function statusLabel(status: Issue['status']): string {
  switch (status) {
    case 'todo':        return 'Todo'
    case 'in_progress': return 'In Progress'
    case 'in_review':   return 'In Review'
    case 'done':        return 'Done'
    case 'backlog':     return 'Backlog'
    default:            return status
  }
}
</script>

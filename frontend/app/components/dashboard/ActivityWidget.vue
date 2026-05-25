<template>
  <div class="card bg-base-100 shadow border border-base-200">
    <div class="card-body">
      <div class="flex items-center justify-between mb-2">
        <h2 class="card-title text-base">Recent Activity</h2>
        <NuxtLink to="/settings" class="text-sm text-primary hover:underline">View all activity →</NuxtLink>
      </div>

      <!-- Skeleton -->
      <div v-if="loading" class="space-y-2">
        <div v-for="i in 5" :key="i" class="skeleton h-6 w-full rounded" />
      </div>

      <!-- Empty state -->
      <p v-else-if="events.length === 0" class="text-base-content/40 text-sm py-4">
        No recent activity yet.
      </p>

      <!-- Timeline -->
      <div v-else class="space-y-1">
        <div
          v-for="event in events"
          :key="event.id"
          class="flex gap-3 items-start py-1 text-sm"
        >
          <span class="text-base-content/40 w-32 shrink-0 text-xs mt-0.5">
            {{ formatDateTime(event.created_at) }}
          </span>
          <span class="text-base-content/70">{{ describeEvent(event) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '~/stores/auth.store'
import type { ActivityEvent } from '~/types/domain.types'
import { ActivityKind } from '~/types/domain.types'

const props = defineProps<{ loading: boolean }>()

const authStore = useAuthStore()
const { formatDateTime } = useTimeFormatter()

// Show last 10 events
const events = computed<ActivityEvent[]>(() => authStore.myActivity.slice(0, 10))

function describeEvent(e: ActivityEvent): string {
  const actor = e.actor_name || `User ${e.actor_id}`
  const issue = e.issue_key ? `[${e.issue_key}] ` : ''
  switch (e.kind) {
    case ActivityKind.StatusChanged:      return `${issue}${actor} changed status from "${e.old_value}" to "${e.new_value}"`
    case ActivityKind.AssigneeChanged:    return `${issue}${actor} changed assignee`
    case ActivityKind.PriorityChanged:    return `${issue}${actor} changed priority from "${e.old_value}" to "${e.new_value}"`
    case ActivityKind.SprintChanged:      return `${issue}${actor} changed sprint`
    case ActivityKind.StoryPointsChanged: return `${issue}${actor} changed story points from ${e.old_value} to ${e.new_value}`
    case ActivityKind.LabelAdded:         return `${issue}${actor} added label "${e.new_value}"`
    case ActivityKind.LabelRemoved:       return `${issue}${actor} removed label "${e.old_value}"`
    case ActivityKind.CommentAdded:       return `${issue}${actor} added a comment`
    default:                              return `${issue}${actor} updated issue`
  }
}
</script>

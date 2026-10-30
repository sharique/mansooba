<template>
  <div class="card bg-base-100 shadow border border-base-200">
    <div class="card-body">
      <h2 class="card-title text-base mb-2">Recent Activity</h2>

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
          <span class="text-base-content/70">{{ describeActivity(event) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '~/stores/auth.store'
import type { ActivityEvent } from '~/types/domain.types'
import { describeActivity } from '~/utils/activityText'

const props = defineProps<{ loading: boolean }>()

const authStore = useAuthStore()
const { formatDateTime } = useTimeFormatter()

// Show last 10 events
const events = computed<ActivityEvent[]>(() => authStore.myActivity.slice(0, 10))
</script>

<script setup lang="ts">
import type { Sprint } from '~/types/domain.types'

const props = defineProps<{
  sprint: Sprint
  projectKey: string
  canManage: boolean
  hasActiveSprint: boolean
}>()

const emit = defineEmits<{
  start: [sprint: Sprint]
  complete: [sprint: Sprint]
  edit: [sprint: Sprint]
  delete: [sprint: Sprint]
}>()

const statusBadge: Record<string, string> = {
  Planning:  'badge-neutral',
  Active:    'badge-success',
  Completed: 'badge-ghost',
}
</script>

<template>
  <div class="card card-bordered bg-base-100 shadow-sm">
    <div class="card-body p-4">
      <div class="flex items-start justify-between gap-2">
        <div class="flex-1 min-w-0">
          <div class="flex items-center gap-2 mb-1">
            <span :class="['badge badge-sm', statusBadge[sprint.status]]">
              {{ sprint.status }}
            </span>
            <h3 class="font-semibold truncate">{{ sprint.name }}</h3>
          </div>
          <p v-if="sprint.goal" class="text-sm text-base-content/60 line-clamp-2">
            {{ sprint.goal }}
          </p>
        </div>

        <div v-if="canManage" class="flex gap-1 shrink-0">
          <button
            v-if="sprint.status === 'Planning' && !hasActiveSprint"
            class="btn btn-xs btn-success"
            @click="emit('start', sprint)"
          >
            Start
          </button>

          <button
            v-if="sprint.status === 'Active'"
            class="btn btn-xs btn-warning"
            @click="emit('complete', sprint)"
          >
            Complete
          </button>

          <button
            v-if="sprint.status !== 'Completed'"
            class="btn btn-xs btn-ghost"
            @click="emit('edit', sprint)"
          >
            Edit
          </button>

          <button
            v-if="sprint.status === 'Planning'"
            class="btn btn-xs btn-error btn-outline"
            @click="emit('delete', sprint)"
          >
            Delete
          </button>
        </div>
      </div>

      <div v-if="sprint.start_date || sprint.end_date" class="text-xs text-base-content/50 mt-1">
        {{ sprint.start_date ?? '?' }} → {{ sprint.end_date ?? '?' }}
      </div>
    </div>
  </div>
</template>

<template>
  <div
    class="card bg-base-100 shadow-sm hover:shadow-md transition-shadow cursor-pointer mb-2"
    @click="navigateTo(`/projects/${projectKey}/issues/${issue.id}`)"
  >
    <div class="card-body p-3">
      <div class="flex justify-between items-start">
        <span class="text-xs text-base-content/50 font-mono">{{ issue.key }}</span>
        <span class="badge badge-sm" :class="priorityClass">{{ issue.priority }}</span>
      </div>
      <p class="text-sm font-medium line-clamp-2">{{ issue.title }}</p>
      <div class="flex justify-between items-center mt-1">
        <select
          class="select select-xs"
          :value="issue.status"
          @change.stop="$emit('statusChanged', issue.id, ($event.target as HTMLSelectElement).value)"
        >
          <option value="backlog">Backlog</option>
          <option value="todo">Todo</option>
          <option value="in_progress">In Progress</option>
          <option value="in_review">In Review</option>
          <option value="done">Done</option>
        </select>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Issue } from '~/types/domain.types'

const props = defineProps<{ issue: Issue; projectKey: string }>()
defineEmits<{ statusChanged: [issueId: number, newStatus: string] }>()

const priorityClass = computed(() => ({
  'badge-error': props.issue.priority === 'critical',
  'badge-warning': props.issue.priority === 'high',
  'badge-info': props.issue.priority === 'medium',
  'badge-ghost': props.issue.priority === 'low',
}))
</script>

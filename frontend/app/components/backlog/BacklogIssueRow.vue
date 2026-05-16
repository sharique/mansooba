<script setup lang="ts">
import type { Issue } from '~/types/domain.types'

const props = defineProps<{
  issue: Issue
  projectKey: string
}>()

const priorityBadge: Record<string, string> = {
  critical: 'badge-error',
  high:     'badge-warning',
  medium:   'badge-info',
  low:      'badge-ghost',
}

const typeIcon: Record<string, string> = {
  epic:  '⚡',
  story: '📖',
  task:  '✓',
  bug:   '🐛',
}
</script>

<template>
  <div
    class="flex items-center gap-3 px-4 py-3 hover:bg-base-200 rounded-lg transition-colors cursor-pointer"
    @click="navigateTo(`/projects/${projectKey}/issues/${issue.id}`)"
  >
    <!-- Issue type icon -->
    <span class="text-base w-5 text-center shrink-0" :title="issue.type">
      {{ typeIcon[issue.type] ?? '·' }}
    </span>

    <!-- Title -->
    <span class="flex-1 text-sm truncate">{{ issue.title }}</span>

    <!-- Story points -->
    <span
      v-if="issue.storyPoints != null"
      class="badge badge-outline badge-sm shrink-0"
      title="Story points"
    >
      {{ issue.storyPoints }}
    </span>

    <!-- Priority badge -->
    <span :class="['badge badge-sm shrink-0', priorityBadge[issue.priority] ?? 'badge-ghost']">
      {{ issue.priority }}
    </span>

    <!-- Assignee initials -->
    <div
      v-if="issue.assigneeId"
      class="avatar placeholder shrink-0"
      title="Assigned"
    >
      <div class="bg-neutral text-neutral-content rounded-full w-6">
        <span class="text-xs">{{ String(issue.assigneeId).slice(0, 2) }}</span>
      </div>
    </div>
  </div>
</template>

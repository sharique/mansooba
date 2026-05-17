<script setup lang="ts">
import type { Issue, Sprint } from '~/types/domain.types'

const props = defineProps<{
  issues: Issue[]
  projectKey: string
  loading?: boolean
  canManage?: boolean
  sprints?: Sprint[]
}>()

const emit = defineEmits<{
  'sprint-assign': [{ issueId: number; sprintId: number }]
}>()
</script>

<template>
  <div>
    <!-- Loading skeleton -->
    <template v-if="loading">
      <div
        v-for="i in 5"
        :key="i"
        class="skeleton h-12 w-full rounded-lg mb-1"
      />
    </template>

    <!-- Empty state -->
    <div
      v-else-if="issues.length === 0"
      class="text-center py-12 text-base-content/50"
    >
      <p class="text-lg">No issues in the backlog</p>
      <p class="text-sm mt-1">Create an issue and leave its sprint unset to have it appear here.</p>
    </div>

    <!-- Issue list -->
    <div v-else class="divide-y divide-base-200 border border-base-200 rounded-lg">
      <BacklogIssueRow
        v-for="issue in issues"
        :key="issue.id"
        :issue="issue"
        :project-key="projectKey"
        :can-manage="canManage"
        :sprints="sprints"
        @sprint-assign="emit('sprint-assign', $event)"
      />
    </div>
  </div>
</template>

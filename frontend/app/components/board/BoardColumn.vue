<template>
  <div class="card bg-base-200 w-72 flex-shrink-0">
    <div class="card-body p-3">
      <div class="flex justify-between items-center mb-2">
        <h3 class="font-semibold capitalize">{{ column.status.replace('_', ' ') }}</h3>
        <span class="badge">{{ column.issues.length }}</span>
      </div>
      <div class="min-h-32">
        <BoardCard
          v-for="issue in column.issues"
          :key="issue.id"
          :issue="issue"
          :project-key="projectKey"
          @status-changed="(id, status) => $emit('issueStatusChanged', id, status)"
        />
      </div>
      <button
        class="btn btn-ghost btn-sm w-full mt-1"
        @click="$emit('createIssue', column.status)"
      >
        + Add Issue
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { BoardColumn } from '~/services/board.service'

defineProps<{ column: BoardColumn; projectKey: string }>()
defineEmits<{
  issueStatusChanged: [issueId: number, newStatus: string]
  createIssue: [status: string]
}>()
</script>

<template>
  <div class="card bg-base-200 w-72 flex-shrink-0 border-t-4" :class="statusColumnBorderClass(column.status)">
    <div class="card-body p-3">
      <div class="flex justify-between items-center mb-3">
        <h3 class="font-semibold capitalize text-sm">{{ column.status.replace(/_/g, ' ') }}</h3>
        <span class="badge badge-sm" :class="statusBadgeClass(column.status)">{{ column.issues.length }}</span>
      </div>
      <div class="min-h-32">
        <BoardCard
          v-for="issue in column.issues"
          :key="issue.id"
          :issue="issue"
          :project-key="projectKey"
        />
      </div>
      <button
        class="btn btn-ghost btn-sm w-full mt-1 opacity-60 hover:opacity-100 transition-opacity"
        @click="$emit('createIssue', column.status)"
      >
        <Icon name="mdi:plus" class="w-4 h-4" />
        Create issue
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { BoardColumn } from '~/services/board.service'
import { statusBadgeClass, statusColumnBorderClass } from '~/utils/issueStyles'

defineProps<{ column: BoardColumn; projectKey: string }>()
defineEmits<{ createIssue: [status: string] }>()
</script>

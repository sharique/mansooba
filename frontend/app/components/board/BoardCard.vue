<template>
  <div
    class="card bg-base-100 shadow-sm hover:shadow-md transition-shadow cursor-pointer mb-2 border-l-4"
    :class="priorityBorderClass(issue.priority)"
    @click="navigateTo(`/projects/${projectKey}/issues/${issue.id}`)"
  >
    <div class="card-body p-3 gap-1.5">
      <!-- Header row: type icon + key -->
      <div class="flex items-center justify-between gap-2">
        <div class="flex items-center gap-1.5 min-w-0">
          <Icon :name="typeIconName(issue.type)" class="w-3.5 h-3.5 shrink-0" :class="typeIconClass(issue.type)" />
          <span class="text-xs text-base-content/45 font-mono shrink-0">{{ issue.key }}</span>
        </div>
        <span class="badge badge-sm shrink-0" :class="priorityBadgeClass(issue.priority)">{{ issue.priority }}</span>
      </div>

      <!-- Title -->
      <p class="text-sm font-medium line-clamp-2">{{ issue.title }}</p>

      <!-- Footer: assignee -->
      <div v-if="issue.assignee_id" class="flex justify-end mt-0.5">
        <UserAvatar
          :avatarUrl="issue.assignee_avatar_url || undefined"
          :name="issue.assignee_name || ''"
          :userId="issue.assignee_id"
          size="sm"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Issue } from '~/types/domain.types'
import { priorityBadgeClass, priorityBorderClass, typeIconName, typeIconClass } from '~/utils/issueStyles'
import UserAvatar from '~/components/common/UserAvatar.vue'

defineProps<{ issue: Issue; projectKey: string }>()
</script>

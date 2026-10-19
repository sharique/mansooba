<template>
  <section class="card bg-base-100 border border-base-300" aria-labelledby="overview-tasks-title">
    <div class="card-body gap-3">
      <div class="flex items-center justify-between gap-2 ">
        <h2 id="overview-tasks-title" class="card-title text-base border-b pb-1">Open tasks</h2>
        <NuxtLink
          v-if="data?.sprint_active"
          :to="`/projects/${projectKey}/board`"
          class="btn btn-accent"
        >Goto board</NuxtLink>
      </div>

      <div v-if="loading" class="space-y-2">
        <div v-for="n in 3" :key="n" class="skeleton h-10 w-full" />
      </div>

      <div v-else-if="error" class="alert alert-error text-sm" role="alert">
        <span class="flex-1">Could not load the open tasks: {{ error }}</span>
        <button class="btn btn-sm" @click="emit('retry')">Retry</button>
      </div>

      <p v-else-if="!data?.sprint_active" class="text-sm text-base-content/60">
        No active sprint, so there are no sprint tasks to show.
      </p>

      <p v-else-if="!data.items.length" class="text-sm text-base-content/60">No open tasks in this sprint.</p>

      <ul v-else class="divide-y divide-base-200">
        <li v-for="t in data.items" :key="t.issue_id" :data-task="t.key" class="py-2">
          <NuxtLink
            :to="`/projects/${projectKey}/issues/${t.issue_id}`"
            class="flex flex-wrap items-center gap-x-3 gap-y-1 hover:bg-base-200 rounded-box px-1"
          >
            <span class="min-w-0 basis-full truncate sm:basis-0 sm:flex-1">
              <span class="font-mono text-xs text-base-content/60">{{ t.key }}</span>
              {{ t.title }}
            </span>
            <span class="badge badge-sm" :class="statusBadgeClass(t.status as never)">{{ statusLabel(t.status as never) }}</span>
            <span class="badge badge-sm" :class="priorityBadgeClass(t.priority as never)">{{ t.priority }}</span>
            <span class="flex items-center gap-1 text-xs text-base-content/70">
              <template v-if="t.assignee">
                <UserAvatar :avatar-url="t.assignee.avatar_url" :name="t.assignee.name" :user-id="t.assignee.id" size="sm" />
                {{ t.assignee.name }}
              </template>
              <template v-else>Unassigned</template>
            </span>
          </NuxtLink>
        </li>
      </ul>
    </div>
  </section>
</template>

<script setup lang="ts">
import UserAvatar from '~/components/common/UserAvatar.vue'
import { priorityBadgeClass, statusBadgeClass, statusLabel } from '~/utils/issueStyles'
import type { OpenTask, OverviewList } from '~/types/domain.types'

defineProps<{
  projectKey: string
  data: OverviewList<OpenTask> | null
  loading: boolean
  error: string | null
}>()
const emit = defineEmits<{ (e: 'retry'): void }>()
</script>

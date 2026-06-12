<template>
  <div class="flex gap-2 items-start py-1 text-sm">
    <UserAvatar
      :avatarUrl="event.actor_avatar_url || undefined"
      :name="event.actor_name || ''"
      :userId="event.actor_id"
      size="sm"
      class="shrink-0 mt-0.5"
    />
    <div class="flex-1 min-w-0">
      <span class="text-base-content/40 text-xs mr-2">{{ formatDateTime(event.created_at) }}</span>
      <span class="text-base-content/70">{{ description }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ActivityEvent } from '~/types/domain.types'
import { ActivityKind } from '~/types/domain.types'
import UserAvatar from '~/components/common/UserAvatar.vue'

const props = defineProps<{ event: ActivityEvent }>()
const { formatDateTime } = useTimeFormatter()

const description = computed(() => {
  const e = props.event
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
})
</script>

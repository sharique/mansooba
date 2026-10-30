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
import { describeActivity } from '~/utils/activityText'
import UserAvatar from '~/components/common/UserAvatar.vue'

const props = defineProps<{ event: ActivityEvent }>()
const { formatDateTime } = useTimeFormatter()

const description = computed(() => describeActivity(props.event))
</script>

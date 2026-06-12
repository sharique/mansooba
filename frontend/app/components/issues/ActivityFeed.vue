<template>
  <div class="space-y-1">
    <template v-for="item in timeline" :key="item.key">
      <ActivityItem
        v-if="item.type === 'activity'"
        :event="(item as ActivityEvent)"
      />
      <div v-else class="flex gap-2 items-start py-1 text-sm">
        <span class="text-base-content/40 w-20 shrink-0 text-xs mt-0.5">{{ formatDateTime(item.created_at) }}</span>
        <span class="text-base-content/70">{{ (item as Comment).author_name || 'Unknown' }} added a comment</span>
      </div>
    </template>
    <p v-if="timeline.length === 0" class="text-base-content/40 text-sm">No activity yet.</p>
  </div>
</template>

<script setup lang="ts">
import type { Comment, ActivityEvent } from '~/types/domain.types'
import { ActivityKind } from '~/types/domain.types'
import ActivityItem from '~/components/issues/ActivityItem.vue'

const props = defineProps<{ comments: Comment[]; activity: ActivityEvent[] }>()

interface TimelineItem { key: string; type: 'activity' | 'comment'; created_at: string }

const timeline = computed<(TimelineItem & (ActivityEvent | Comment))[]>(() => {
  const acts = props.activity
    .filter(a => a.kind !== ActivityKind.CommentAdded)
    .map(a => ({ ...a, key: `a-${a.id}`, type: 'activity' as const }))
  const coms = props.comments.map(c => ({ ...c, key: `c-${c.id}`, type: 'comment' as const }))
  return [...acts, ...coms].sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime())
})

const { formatDateTime } = useTimeFormatter()
</script>

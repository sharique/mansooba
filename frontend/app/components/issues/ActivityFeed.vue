<template>
  <div class="space-y-1">
    <div v-for="item in timeline" :key="item.key" class="flex gap-2 items-start py-1 text-sm">
      <span class="text-base-content/40 w-20 shrink-0 text-xs mt-0.5">{{ formatDate(item.created_at) }}</span>
      <span v-if="item.type === 'activity'" class="text-base-content/70">{{ describeEvent(item as ActivityEvent) }}</span>
      <span v-else class="text-base-content/70">User {{ (item as Comment).author_id }} added a comment</span>
    </div>
    <p v-if="timeline.length === 0" class="text-base-content/40 text-sm">No activity yet.</p>
  </div>
</template>

<script setup lang="ts">
import type { Comment, ActivityEvent } from '~/types/domain.types'
import { ActivityKind } from '~/types/domain.types'

const props = defineProps<{ comments: Comment[]; activity: ActivityEvent[] }>()

interface TimelineItem { key: string; type: 'activity' | 'comment'; created_at: string }

const timeline = computed<(TimelineItem & (ActivityEvent | Comment))[]>(() => {
  const acts = props.activity.map(a => ({ ...a, key: `a-${a.id}`, type: 'activity' as const }))
  const coms = props.comments.map(c => ({ ...c, key: `c-${c.id}`, type: 'comment' as const }))
  return [...acts, ...coms].sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime())
})

function formatDate(iso: string) { return new Date(iso).toLocaleDateString() }

function describeEvent(e: ActivityEvent): string {
  switch (e.kind) {
    case ActivityKind.StatusChanged:      return `Status changed from "${e.old_value}" to "${e.new_value}"`
    case ActivityKind.AssigneeChanged:    return `Assignee changed from "${e.old_value}" to "${e.new_value}"`
    case ActivityKind.PriorityChanged:    return `Priority changed from "${e.old_value}" to "${e.new_value}"`
    case ActivityKind.SprintChanged:      return `Sprint changed from "${e.old_value}" to "${e.new_value}"`
    case ActivityKind.StoryPointsChanged: return `Story points changed from ${e.old_value} to ${e.new_value}`
    case ActivityKind.LabelAdded:         return `Label "${e.new_value}" added`
    case ActivityKind.LabelRemoved:       return `Label "${e.old_value}" removed`
    default:                              return `Updated`
  }
}
</script>

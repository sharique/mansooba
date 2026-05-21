<template>
  <div class="space-y-1">
    <div v-for="item in timeline" :key="item.key" class="flex gap-2 items-start py-1 text-sm">
      <span class="text-base-content/40 w-20 shrink-0 text-xs mt-0.5">{{ formatDate(item.created_at) }}</span>
      <span v-if="item.type === 'activity'" class="text-base-content/70">{{ describeEvent(item as ActivityEvent) }}</span>
      <span v-else class="text-base-content/70">{{ (item as Comment).author_name || 'Unknown' }} added a comment</span>
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
  const acts = props.activity
    .filter(a => a.kind !== ActivityKind.CommentAdded)
    .map(a => ({ ...a, key: `a-${a.id}`, type: 'activity' as const }))
  const coms = props.comments.map(c => ({ ...c, key: `c-${c.id}`, type: 'comment' as const }))
  return [...acts, ...coms].sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime())
})

function formatDate(iso: string) { return new Date(iso).toLocaleDateString() }

function describeEvent(e: ActivityEvent): string {
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
}
</script>

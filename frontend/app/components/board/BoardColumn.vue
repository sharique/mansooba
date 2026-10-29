<template>
  <!-- The column rings itself while a dragged card hovers over it: the drag library keeps a
       ghost copy (.sortable-ghost) in whichever list the card is currently over. -->
  <div
    class="card bg-base-200 w-72 flex-shrink-0 border-t-4 has-[.sortable-ghost]:ring-2 has-[.sortable-ghost]:ring-primary"
    :class="statusColumnBorderClass(column.status)"
  >
    <div class="card-body p-3">
      <div class="flex justify-between items-center mb-3">
        <h3 class="font-semibold capitalize text-sm">{{ column.status.replace(/_/g, ' ') }}</h3>
        <span class="badge badge-sm" :class="statusBadgeClass(column.status)">{{ column.issues.length }}</span>
      </div>
      <!-- The list is driven by the store (read-only model, no v-model), so a move is only
           ever applied by the store and a failed save can put the card back. -->
      <VueDraggable
        :model-value="column.issues"
        group="board"
        :sort="false"
        filter=".no-drag"
        :prevent-on-filter="false"
        :animation="150"
        ghost-class="opacity-40"
        :delay="150"
        :delay-on-touch-only="true"
        :touch-start-threshold="4"
        class="min-h-32"
        @add="onAdd"
        @end="markDragEnd()"
      >
        <BoardCard
          v-for="issue in column.issues"
          :key="issue.id"
          :issue="issue"
          :project-key="projectKey"
          @move="emit('moved', $event)"
        />
      </VueDraggable>
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
import { VueDraggable } from 'vue-draggable-plus'
import type { BoardColumn } from '~/services/board.service'
import { markDragEnd } from '~/utils/dragGuard'
import { statusBadgeClass, statusColumnBorderClass } from '~/utils/issueStyles'
import type { Issue } from '~/types/domain.types'

const props = defineProps<{ column: BoardColumn; projectKey: string }>()
const emit = defineEmits<{
  createIssue: [status: string]
  moved: [move: { issueId: number; toStatus: Issue['status'] }]
}>()

// The part of the drag library's event this column reads.
interface AddEvent {
  item: HTMLElement
  from: HTMLElement
  oldIndex?: number
  originalEvent?: { dataTransfer?: { dropEffect?: string } }
}

// A card was dropped on this column. The library also reports a drag cancelled with Escape
// as a drop, with dropEffect 'none' (a touch drag has no dataTransfer and is a real drop).
// A cancelled card has already been moved in the DOM, so put it back where it was and
// change nothing.
function onAdd(evt: AddEvent) {
  if (evt.originalEvent?.dataTransfer?.dropEffect === 'none') {
    evt.from.insertBefore(evt.item, evt.from.children[evt.oldIndex ?? 0] ?? null)
    return
  }
  emit('moved', { issueId: Number(evt.item.dataset.issueId), toStatus: props.column.status as Issue['status'] })
}
</script>

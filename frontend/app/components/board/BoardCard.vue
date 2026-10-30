<template>
  <div
    class="card bg-base-100 shadow-sm hover:shadow-md transition-shadow cursor-pointer mb-2 border-l-4 touch-manipulation"
    :class="[priorityBorderClass(issue.priority), { 'no-drag': issue.locked }]"
    :aria-disabled="issue.locked ? 'true' : undefined"
    :data-issue-id="issue.id"
    @click="open"
  >
    <div class="card-body p-3 gap-1.5">
      <!-- Header row: type icon + key -->
      <div class="flex items-center justify-between gap-2">
        <div class="flex items-center gap-1.5 min-w-0">
          <Icon :name="typeIconName(issue.type)" class="w-3.5 h-3.5 shrink-0" :class="typeIconClass(issue.type)" />
          <span class="text-xs text-base-content/45 font-mono shrink-0">{{ issue.key }}</span>
        </div>
        <div class="flex items-center gap-1 shrink-0">
          <span class="badge badge-sm" :class="priorityBadgeClass(issue.priority)">{{ issue.priority }}</span>
          <!-- Keyboard and screen-reader way to move the card: a disclosure button with a
               list of column buttons. Dragging is not needed for anything. -->
          <div ref="moveEl" class="relative" @click.stop @focusout="onFocusOut" @keydown.esc="onEscape">
            <button
              ref="toggleEl"
              type="button"
              class="btn btn-ghost btn-xs btn-square"
              :aria-label="`Move ${issue.key} to…`"
              :aria-expanded="String(menuOpen)"
              :aria-controls="listId"
              :aria-disabled="issue.locked ? 'true' : undefined"
              :title="issue.locked ? 'Sprint completed' : undefined"
              @click="toggleMenu"
            >
              <Icon name="mdi:dots-horizontal" class="w-4 h-4" />
            </button>
            <ul
              v-if="menuOpen"
              :id="listId"
              class="menu bg-base-100 rounded-box shadow-lg border border-base-300 absolute right-0 z-20 w-40 p-1"
            >
              <li v-for="s in targets" :key="s">
                <button type="button" @click="choose(s)">{{ statusLabel(s) }}</button>
              </li>
            </ul>
          </div>
        </div>
      </div>

      <!-- Title -->
      <p class="text-sm font-medium line-clamp-2">{{ issue.title }}</p>

      <!-- Locked: the task belongs to a completed sprint, so it cannot be moved from here -->
      <div v-if="issue.locked" class="flex items-center gap-1 text-xs text-base-content/50" title="Sprint completed">
        <Icon name="mdi:lock-outline" class="w-3.5 h-3.5" />
        Sprint completed
      </div>

      <!-- Footer: attachment count + assignee -->
      <div v-if="issue.attachment_count > 0 || issue.assignee_id" class="flex items-center justify-between mt-0.5">
        <div v-if="issue.attachment_count > 0" class="flex items-center gap-1 text-xs text-base-content/45">
          <Icon name="mdi:paperclip" class="w-3.5 h-3.5" />
          {{ issue.attachment_count }}
        </div>
        <div v-else />
        <UserAvatar
          v-if="issue.assignee_id"
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
import { BOARD_STATUSES, priorityBadgeClass, priorityBorderClass, statusLabel, typeIconName, typeIconClass } from '~/utils/issueStyles'
import UserAvatar from '~/components/common/UserAvatar.vue'
import { wasJustDragged } from '~/utils/dragGuard'

const props = defineProps<{ issue: Issue; projectKey: string }>()
const emit = defineEmits<{ move: [move: { issueId: number; toStatus: Issue['status'] }] }>()

const menuOpen = ref(false)
const moveEl = ref<HTMLElement | null>(null)
const toggleEl = ref<HTMLButtonElement | null>(null)
const listId = computed(() => `move-list-${props.issue.id}`)
const targets = computed(() => BOARD_STATUSES.filter(s => s !== props.issue.status))

function toggleMenu() {
  if (props.issue.locked) return
  menuOpen.value = !menuOpen.value
}

function closeMenu(restoreFocus: boolean) {
  menuOpen.value = false
  if (restoreFocus) nextTick(() => toggleEl.value?.focus())
}

// Escape works from the button as well as from inside the list: opening the list leaves
// the focus on the button.
function onEscape(e: KeyboardEvent) {
  if (!menuOpen.value) return
  e.stopPropagation()
  closeMenu(true)
}

function choose(toStatus: Issue['status']) {
  closeMenu(false)
  emit('move', { issueId: props.issue.id, toStatus })
}

// Tabbing out of the control (or clicking elsewhere) closes the list.
function onFocusOut(e: FocusEvent) {
  const next = e.relatedTarget as Node | null
  if (menuOpen.value && !(next && moveEl.value?.contains(next))) menuOpen.value = false
}

// A click right after a drag ends is part of the drag, not a request to open the task.
function open() {
  if (wasJustDragged()) return
  navigateTo(`/projects/${props.projectKey}/issues/${props.issue.id}`)
}
</script>

<template>
  <div class="flex gap-3 py-3 border-b border-base-200 last:border-0">
    <UserAvatar
      :avatarUrl="comment.author_avatar_url || undefined"
      :name="comment.author_name || ''"
      :userId="comment.author_id"
      size="sm"
    />
    <div class="flex-1 min-w-0">
      <div class="flex items-baseline gap-2 mb-1">
        <span class="font-medium text-sm">{{ comment.author_name }}</span>
        <span class="text-xs text-base-content/50">{{ relativeTime }}</span>
      </div>
      <div v-if="editing">
        <textarea v-model="editBody" class="textarea textarea-bordered w-full text-sm" rows="3" />
        <div class="flex gap-2 mt-1">
          <button class="btn btn-xs btn-primary" :disabled="!editBody.trim()" @click="saveEdit">Save</button>
          <button class="btn btn-xs btn-ghost" @click="editing = false">Cancel</button>
        </div>
      </div>
      <div v-else class="prose prose-sm max-w-none" v-html="rendered" />
      <div v-if="isOwn && !editing" class="flex gap-2 mt-1">
        <button class="btn btn-xs btn-ghost" @click="startEdit">Edit</button>
        <button class="btn btn-xs btn-ghost text-error" @click="$emit('delete', comment.id)">Delete</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Comment } from '~/types/domain.types'
import UserAvatar from '~/components/common/UserAvatar.vue'

const props = defineProps<{ comment: Comment; currentUserId: number }>()
const emit = defineEmits<{ (e: 'update', id: number, body: string): void; (e: 'delete', id: number): void }>()

const editing = ref(false)
const editBody = ref(props.comment.body)

const isOwn = computed(() => props.comment.author_id === props.currentUserId)
const rendered = computed(() => useMarkdown(props.comment.body))
const { formatDateTime } = useTimeFormatter()
const relativeTime = computed(() => formatDateTime(props.comment.created_at))

function startEdit() {
  editBody.value = props.comment.body
  editing.value = true
}
function saveEdit() {
  if (!editBody.value.trim()) return
  emit('update', props.comment.id, editBody.value)
  editing.value = false
}
</script>

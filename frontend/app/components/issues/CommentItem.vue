<template>
  <div class="flex gap-3 py-3 border-b border-base-200 last:border-0">
    <div class="avatar placeholder">
      <div class="bg-neutral text-neutral-content rounded-full w-8 h-8">
        <span class="text-xs">{{ initials }}</span>
      </div>
    </div>
    <div class="flex-1 min-w-0">
      <div class="flex items-baseline gap-2 mb-1">
        <span class="font-medium text-sm">User {{ comment.author_id }}</span>
        <span class="text-xs text-base-content/50">{{ relativeTime }}</span>
      </div>
      <div v-if="editing">
        <textarea v-model="editBody" class="textarea textarea-bordered w-full text-sm" rows="3" />
        <div class="flex gap-2 mt-1">
          <button class="btn btn-xs btn-primary" @click="saveEdit">Save</button>
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

const props = defineProps<{ comment: Comment; currentUserId: number }>()
const emit = defineEmits<{ (e: 'update', id: number, body: string): void; (e: 'delete', id: number): void }>()

const editing = ref(false)
const editBody = ref(props.comment.body)

const isOwn = computed(() => props.comment.author_id === props.currentUserId)
const rendered = computed(() => useMarkdown(props.comment.body))
const initials = computed(() => String(props.comment.author_id).slice(0, 2))
const relativeTime = computed(() => new Date(props.comment.created_at).toLocaleDateString())

function startEdit() {
  editBody.value = props.comment.body
  editing.value = true
}
function saveEdit() {
  emit('update', props.comment.id, editBody.value)
  editing.value = false
}
</script>

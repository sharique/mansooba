<template>
  <div>
    <h3 class="font-semibold text-sm mb-3">Comments</h3>
    <div v-if="store.loading" class="loading loading-spinner loading-sm" />
    <div v-else>
      <IssuesCommentItem
        v-for="c in store.comments"
        :key="c.id"
        :comment="c"
        :current-user-id="currentUserId"
        @update="handleUpdate"
        @delete="handleDelete"
      />
    </div>
    <div class="mt-4">
      <textarea
        v-model="newBody"
        class="textarea textarea-bordered w-full text-sm"
        rows="3"
        placeholder="Add a comment… (supports **markdown**)"
      />
      <button class="btn btn-sm btn-primary mt-2" :disabled="!newBody.trim()" @click="submit">
        Comment
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useCommentsStore } from '~/stores/comments.store'

const props = defineProps<{ issueId: number; currentUserId: number }>()
const store = useCommentsStore()
const newBody = ref('')

onMounted(() => store.fetchComments(props.issueId))

function handleUpdate(id: number, body: string) {
  store.updateComment(props.issueId, id, body)
}

function handleDelete(id: number) {
  store.deleteComment(props.issueId, id)
}

async function submit() {
  if (!newBody.value.trim()) return
  await store.addComment(props.issueId, newBody.value.trim())
  newBody.value = ''
}
</script>

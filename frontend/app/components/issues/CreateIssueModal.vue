<template>
  <dialog ref="dialogEl" class="modal">
    <div class="modal-box">
      <h3 class="font-bold text-lg mb-4">New Issue</h3>

      <IssuesIssueForm
        :project-key="props.projectKey"
        :default-status="props.defaultStatus ?? 'backlog'"
        @saved="onSaved"
        @cancelled="$emit('close')"
      />
    </div>
    <form method="dialog" class="modal-backdrop"><button @click="$emit('close')">close</button></form>
  </dialog>
</template>

<script setup lang="ts">
import type { Issue } from '~/types/domain.types'

const props = defineProps<{
  projectKey: string
  defaultStatus?: string
  open: boolean
}>()
const emit = defineEmits<{ created: [issue: Issue]; close: [] }>()

const dialogEl = ref<HTMLDialogElement | null>(null)

watch(() => props.open, async (val) => {
  if (val) {
    await nextTick()
    dialogEl.value?.showModal()
  }
  else {
    dialogEl.value?.close()
  }
}, { immediate: true })

function onSaved(issue: Issue) {
  emit('created', issue)
  emit('close')
}
</script>

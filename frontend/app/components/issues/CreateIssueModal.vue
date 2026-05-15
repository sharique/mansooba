<template>
  <dialog ref="dialogEl" class="modal">
    <div class="modal-box">
      <h3 class="font-bold text-lg mb-4">New Issue</h3>
      <IssuesIssueForm
        :project-key="projectKey"
        :default-status="defaultStatus"
        @saved="onSaved"
        @cancelled="$emit('close')"
      />
    </div>
    <form method="dialog" class="modal-backdrop"><button @click="$emit('close')">close</button></form>
  </dialog>
</template>

<script setup lang="ts">
import type { Issue } from '~/types/domain.types'

const props = defineProps<{ projectKey: string; defaultStatus: string; open: boolean }>()
const emit = defineEmits<{ created: [issue: Issue]; close: [] }>()

const dialogEl = ref<HTMLDialogElement | null>(null)

watch(() => props.open, (val) => {
  if (val) dialogEl.value?.showModal()
  else dialogEl.value?.close()
})

function onSaved(issue: Issue) {
  emit('created', issue)
  emit('close')
}
</script>

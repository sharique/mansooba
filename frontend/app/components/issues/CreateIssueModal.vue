<template>
  <dialog ref="dialogEl" class="modal">
    <div class="modal-box">
      <h3 class="font-bold text-lg mb-4">New Issue</h3>

      <!-- Project selector shown when no project context is provided -->
      <div v-if="!props.projectKey" class="form-control mb-4">
        <label class="label"><span class="label-text">Project</span></label>
        <select v-model="selectedProjectKey" class="select select-bordered" required>
          <option value="" disabled>Select a project…</option>
          <option v-for="p in projectsStore.projects" :key="p.id" :value="p.key">
            {{ p.name }} ({{ p.key }})
          </option>
        </select>
      </div>

      <IssuesIssueForm
        v-if="resolvedProjectKey"
        :project-key="resolvedProjectKey"
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
import { useProjectsStore } from '~/stores/projects.store'

const props = defineProps<{
  projectKey?: string
  defaultStatus?: string
  open: boolean
}>()
const emit = defineEmits<{ created: [issue: Issue]; close: [] }>()

const projectsStore = useProjectsStore()
const dialogEl = ref<HTMLDialogElement | null>(null)
const selectedProjectKey = ref('')

const resolvedProjectKey = computed(() =>
  props.projectKey ?? (selectedProjectKey.value || undefined),
)

watch(() => props.open, (val) => {
  if (val) {
    selectedProjectKey.value = ''
    dialogEl.value?.showModal()
  }
  else {
    dialogEl.value?.close()
  }
})

function onSaved(issue: Issue) {
  emit('created', issue)
  emit('close')
}
</script>

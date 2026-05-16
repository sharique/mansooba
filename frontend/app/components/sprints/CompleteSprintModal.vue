<script setup lang="ts">
import type { Sprint } from '~/types/domain.types'

const props = defineProps<{
  projectKey: string
  sprint: Sprint
  otherSprints: Sprint[]
}>()

const emit = defineEmits<{
  completed: [sprint: Sprint]
  cancel: []
}>()

const sprintsStore = useSprintsStore()
const { showSuccess, showError } = useToast()

const selectedNextSprintId = ref<string>('')
const submitting = ref(false)

async function confirm() {
  submitting.value = true
  try {
    const payload = selectedNextSprintId.value
      ? { next_sprint_id: selectedNextSprintId.value }
      : {}
    const updated = await sprintsStore.completeSprint(props.projectKey, props.sprint.id, payload)
    showSuccess(`"${props.sprint.name}" completed`)
    emit('completed', updated)
  }
  catch (e: any) {
    showError(e.data?.message ?? 'Failed to complete sprint')
  }
  finally {
    submitting.value = false
  }
}
</script>

<template>
  <dialog class="modal modal-open">
    <div class="modal-box">
      <h3 class="font-bold text-lg mb-2">Complete "{{ sprint.name }}"</h3>
      <p class="text-base-content/70 text-sm mb-4">
        Where should unfinished issues go?
      </p>

      <label class="form-control">
        <div class="label"><span class="label-text">Move unfinished issues to</span></div>
        <select v-model="selectedNextSprintId" class="select select-bordered">
          <option value="">Backlog (no sprint)</option>
          <option
            v-for="s in otherSprints"
            :key="s.id"
            :value="s.id"
          >
            {{ s.name }} ({{ s.status }})
          </option>
        </select>
      </label>

      <div class="modal-action mt-4">
        <button class="btn btn-ghost" @click="emit('cancel')">Cancel</button>
        <button class="btn btn-warning" :disabled="submitting" @click="confirm">
          <span v-if="submitting" class="loading loading-spinner loading-sm" />
          Complete Sprint
        </button>
      </div>
    </div>
    <div class="modal-backdrop" @click="emit('cancel')" />
  </dialog>
</template>

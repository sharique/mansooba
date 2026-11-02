<template>
  <div>
    <button type="button" class="btn btn-error btn-outline btn-sm" @click="openDialog">Delete project</button>

    <dialog ref="dialogEl" class="modal" aria-labelledby="delete-project-title">
      <div class="modal-box">
        <h3 id="delete-project-title" class="font-bold text-lg mb-2">Delete {{ projectName }}?</h3>
        <p class="text-base-content/70 mb-4">
          This cannot be undone. A project can only be deleted when it has no tasks left.
        </p>
        <label class="form-control">
          <span class="label-text mb-1">
            Type <strong>{{ projectName }}</strong> to confirm
          </span>
          <input
            v-model="typed"
            type="text"
            class="input input-bordered w-full"
            aria-label="Type the project name to confirm"
            autocomplete="off"
            @keyup.enter="matches && submit()"
          >
        </label>

        <div v-if="errorMessage" class="alert alert-error mt-3 py-2 text-sm" role="alert">
          <span data-testid="delete-error">{{ errorMessage }}</span>
        </div>
        <p v-if="taskCount !== null" class="text-sm mt-2" data-testid="task-count">
          Tasks remaining: <strong>{{ taskCount }}</strong>
        </p>

        <div class="flex gap-2 justify-end mt-4">
          <button type="button" class="btn btn-ghost" @click="closeDialog">Cancel</button>
          <button
            type="button"
            class="btn btn-error"
            data-testid="confirm-delete"
            :disabled="!matches || deleting"
            :title="matches ? undefined : 'Type the exact project name first'"
            @click="submit"
          >
            <span v-if="deleting" class="loading loading-spinner loading-sm" />
            Delete project
          </button>
        </div>
      </div>
      <form method="dialog" class="modal-backdrop"><button>close</button></form>
    </dialog>
  </div>
</template>

<script setup lang="ts">
import { ApiMessageError } from '~/utils/apiError'

const props = defineProps<{ projectKey: string; projectName: string }>()

const projectsStore = useProjectsStore()
const { showSuccess } = useToast()

const dialogEl = ref<HTMLDialogElement | null>(null)
const typed = ref('')
const deleting = ref(false)
const errorMessage = ref('')
const taskCount = ref<number | null>(null)

const matches = computed(() => typed.value === props.projectName)

function openDialog() {
  typed.value = ''
  errorMessage.value = ''
  taskCount.value = null
  dialogEl.value?.showModal?.()
}

function closeDialog() {
  dialogEl.value?.close?.()
}

async function submit() {
  if (!matches.value || deleting.value) return
  deleting.value = true
  errorMessage.value = ''
  taskCount.value = null
  try {
    await projectsStore.deleteProject(props.projectKey, typed.value)
    closeDialog()
    showSuccess(`Project "${props.projectName}" deleted`)
    await navigateTo('/projects')
  }
  catch (e) {
    errorMessage.value = (e as Error).message
    if (e instanceof ApiMessageError && typeof e.taskCount === 'number') taskCount.value = e.taskCount
  }
  finally {
    deleting.value = false
  }
}
</script>
